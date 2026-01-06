package virtual_key_middleware

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("Middleware")

// RateLimiterStorage is an interface for storing and retrieving token bucket state.
// This abstraction allows for easy migration from in-memory to Redis-based distributed rate limiting.
type RateLimiterStorage interface {
	// Allow checks if a request is allowed and consumes a token if available.
	// Returns true if allowed, false if rate limited, and any error that occurred.
	Allow(ctx context.Context, virtualKeyID string, rateLimit gateway.RateLimit) (allowed bool, err error)
}

type VirtualKeyMiddleware struct {
	configStore        gateway.ConfigStore
	rateLimiterStorage RateLimiterStorage
}

// NewVirtualKeyMiddleware creates a new VirtualKeyMiddleware with in-memory rate limiting.
// To use Redis-based distributed rate limiting, pass a RedisRateLimiterStorage implementation
// as the optional rateLimiterStorage parameter.
// If no rateLimiterStorage is provided, an in-memory implementation will be used.
func NewVirtualKeyMiddleware(configStore gateway.ConfigStore, rateLimiterStorage ...RateLimiterStorage) *VirtualKeyMiddleware {
	var storage RateLimiterStorage
	if len(rateLimiterStorage) > 0 && rateLimiterStorage[0] != nil {
		storage = rateLimiterStorage[0]
	} else {
		storage = NewInMemoryRateLimiterStorage()
	}

	return &VirtualKeyMiddleware{
		configStore:        configStore,
		rateLimiterStorage: storage,
	}
}

func (middleware *VirtualKeyMiddleware) HandleRequest(next gateway.RequestHandler) gateway.RequestHandler {
	return func(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.Response, error) {
		ctx, span := tracer.Start(ctx, "Middleware.VirtualKeyMiddleware")
		defer span.End()

		// Check rate limits if this is a virtual key
		if strings.HasPrefix(key, "sk-amg") {
			if err := middleware.checkRateLimits(ctx, key); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}
		}

		key, err := middleware.getDirectKey(ctx, providerName, key, r)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		return next(ctx, providerName, key, r)
	}
}

func (middleware *VirtualKeyMiddleware) HandleStreamingRequest(next gateway.StreamingRequestHandler) gateway.StreamingRequestHandler {
	return func(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.StreamingResponse, error) {
		ctx, span := tracer.Start(ctx, "Middleware.VirtualKeyMiddleware")
		defer span.End()

		// Check rate limits if this is a virtual key
		if strings.HasPrefix(key, "sk-amg") {
			if err := middleware.checkRateLimits(ctx, key); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}
		}

		key, err := middleware.getDirectKey(ctx, providerName, key, r)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		return next(ctx, providerName, key, r)
	}
}

func (middleware *VirtualKeyMiddleware) getDirectKey(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (string, error) {
	span := trace.SpanFromContext(ctx)
	model := r.GetRequestedModel()

	// If the virtual key is provided, fetch the associated direct key
	if strings.HasPrefix(key, "sk-amg") {
		if middleware.configStore == nil {
			err := errors.New("config store is required when virtual key is provided")
			return key, err
		}
	} else {
		return key, nil
	}

	virtualKey, err := middleware.configStore.GetVirtualKey(key)
	if err != nil {
		return key, err
	}

	span.SetAttributes(
		attribute.Int("virtual_key.allowed_providers", len(virtualKey.AllowedProviders)),
		attribute.Int("virtual_key.allowed_models", len(virtualKey.AllowedModels)),
	)

	// Check whether the given provider is allowed for the virtual key
	if len(virtualKey.AllowedProviders) > 0 && !slices.Contains(virtualKey.AllowedProviders, providerName) {
		err := errors.New("provider access denied by virtual key")
		return key, err
	}

	// Check whether the given model is allowed for the virtual key
	if len(virtualKey.AllowedModels) > 0 && !slices.Contains(virtualKey.AllowedModels, model) {
		err := errors.New("model access denied by virtual key")
		return key, err
	}

	providerConfig, err := middleware.configStore.GetProviderConfig(providerName)
	if err != nil {
		err = errors.New("failed to get provider config")
		return key, err
	}

	if providerConfig == nil || len(providerConfig.ApiKeys) == 0 {
		err := errors.New("provider configs or api keys are not found")
		return key, err
	}

	if len(providerConfig.ApiKeys) == 1 {
		return providerConfig.ApiKeys[0].APIKey, nil
	}

	// Weight random selection
	weights := make([]int, len(providerConfig.ApiKeys))
	for idx, key := range providerConfig.ApiKeys {
		weights[idx] = key.Weight
	}

	return providerConfig.ApiKeys[utils.WeightedRandomIndex(weights)].APIKey, nil
}

// checkRateLimits validates all rate limits for a virtual key.
// Returns an error if any rate limit is exceeded.
func (middleware *VirtualKeyMiddleware) checkRateLimits(ctx context.Context, virtualKey string) error {
	if middleware.rateLimiterStorage == nil {
		return nil // No rate limiting if storage is not configured
	}

	// Get virtual key config to access rate limits
	vkConfig, err := middleware.configStore.GetVirtualKey(virtualKey)
	if err != nil {
		return err
	}

	// If no rate limits are configured, allow the request
	if len(vkConfig.RateLimits) == 0 {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("virtual_key.rate_limits.count", len(vkConfig.RateLimits)))

	// Check each rate limit
	for _, rateLimit := range vkConfig.RateLimits {
		allowed, err := middleware.rateLimiterStorage.Allow(ctx, virtualKey, rateLimit)
		if err != nil {
			return fmt.Errorf("rate limit check failed: %w", err)
		}

		if !allowed {
			span.SetAttributes(
				attribute.String("rate_limit.unit", rateLimit.Unit),
				attribute.Int64("rate_limit.limit", rateLimit.Limit),
			)
			return fmt.Errorf("rate limit exceeded: %d requests per %s", rateLimit.Limit, rateLimit.Unit)
		}
	}

	return nil
}
