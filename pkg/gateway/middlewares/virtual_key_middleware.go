package middlewares

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
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

// InMemoryRateLimiterStorage implements RateLimiterStorage using in-memory token buckets.
// This can be easily replaced with a Redis-based implementation for distributed rate limiting.
type InMemoryRateLimiterStorage struct {
	mu          sync.RWMutex
	buckets     map[string]*tokenBucket
	cleanup     *time.Ticker
	stopCleanup chan struct{}
}

// tokenBucket represents a single token bucket for rate limiting.
type tokenBucket struct {
	mu             sync.Mutex
	tokens         float64
	lastRefill     time.Time
	capacity       float64
	refillRate     float64 // tokens per second
	windowDuration time.Duration
}

// NewInMemoryRateLimiterStorage creates a new in-memory rate limiter storage.
// It includes a background cleanup goroutine to remove unused buckets.
func NewInMemoryRateLimiterStorage() *InMemoryRateLimiterStorage {
	storage := &InMemoryRateLimiterStorage{
		buckets:     make(map[string]*tokenBucket),
		cleanup:     time.NewTicker(5 * time.Minute),
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go storage.cleanupUnusedBuckets()

	return storage
}

// Stop stops the background cleanup goroutine. Call this when shutting down.
func (s *InMemoryRateLimiterStorage) Stop() {
	s.cleanup.Stop()
	close(s.stopCleanup)
}

// Allow checks if a request is allowed and consumes a token if available.
func (s *InMemoryRateLimiterStorage) Allow(ctx context.Context, virtualKeyID string, rateLimit gateway.RateLimit) (bool, error) {
	bucketKey := s.getBucketKey(virtualKeyID, rateLimit.Unit)

	// Parse the rate limit unit to get duration
	duration, err := parseRateLimitUnit(rateLimit.Unit)
	if err != nil {
		return false, fmt.Errorf("invalid rate limit unit: %w", err)
	}

	// Get or create bucket
	s.mu.Lock()
	bucket, exists := s.buckets[bucketKey]
	if !exists {
		bucket = s.newTokenBucket(float64(rateLimit.Limit), duration)
		s.buckets[bucketKey] = bucket
	}
	s.mu.Unlock()

	// Try to consume a token
	return bucket.consume(1), nil
}

// getBucketKey generates a unique key for a virtual key + rate limit unit combination.
func (s *InMemoryRateLimiterStorage) getBucketKey(virtualKeyID string, unit string) string {
	return fmt.Sprintf("%s:%s", virtualKeyID, unit)
}

// newTokenBucket creates a new token bucket with the given capacity and window duration.
func (s *InMemoryRateLimiterStorage) newTokenBucket(capacity float64, windowDuration time.Duration) *tokenBucket {
	now := time.Now()
	// Refill rate is capacity / window duration in seconds
	refillRate := capacity / windowDuration.Seconds()

	return &tokenBucket{
		tokens:         capacity,
		lastRefill:     now,
		capacity:       capacity,
		refillRate:     refillRate,
		windowDuration: windowDuration,
	}
}

// consume attempts to consume the requested number of tokens.
// Returns true if tokens were available and consumed, false otherwise.
func (tb *tokenBucket) consume(tokens float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// Refill tokens based on elapsed time
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.refillRate
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	// Check if we have enough tokens
	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}

	return false
}

// cleanupUnusedBuckets periodically removes buckets that haven't been used recently.
func (s *InMemoryRateLimiterStorage) cleanupUnusedBuckets() {
	for {
		select {
		case <-s.cleanup.C:
			s.mu.Lock()
			now := time.Now()
			for key, bucket := range s.buckets {
				bucket.mu.Lock()
				// Remove buckets that haven't been used in 2x their window duration
				unusedDuration := now.Sub(bucket.lastRefill)
				if unusedDuration > bucket.windowDuration*2 {
					delete(s.buckets, key)
				}
				bucket.mu.Unlock()
			}
			s.mu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}

// parseRateLimitUnit converts a rate limit unit string to a time.Duration.
// Supported units: 1min, 1h, 6h, 12h, 1d, 1w, 1mo
func parseRateLimitUnit(unit string) (time.Duration, error) {
	switch unit {
	case "1min":
		return time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "1w":
		return 7 * 24 * time.Hour, nil
	case "1mo":
		return 30 * 24 * time.Hour, nil // Approximate month as 30 days
	default:
		return 0, fmt.Errorf("unsupported rate limit unit: %s", unit)
	}
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

	// Todo: support for key rotation
	return providerConfig.ApiKeys[0].APIKey, nil
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
