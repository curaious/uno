package middlewares

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("Middleware")

type VirtualKeyMiddleware struct {
	configStore gateway.ConfigStore
}

func NewVirtualKeyMiddleware(configStore gateway.ConfigStore) *VirtualKeyMiddleware {
	return &VirtualKeyMiddleware{
		configStore: configStore,
	}
}

func (middleware *VirtualKeyMiddleware) HandleRequest(next gateway.RequestHandler) gateway.RequestHandler {
	return func(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.Response, error) {
		ctx, span := tracer.Start(ctx, "Middleware.VirtualKeyMiddleware")
		defer span.End()

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
