package gateway

import (
	"context"
	"errors"

	"github.com/praveen001/uno/pkg/llm"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("LLMGateway")

// ConfigStore is the interface required by LLMGateway to get provider and virtual key configurations.
// Implementations:
// - InMemoryConfigStore: for SDK consumers with direct API keys
// - ServiceConfigStore (in gateway_adapter): for server-side use with database-backed configs
// - ExternalConfigStore (in gateway_adapter): for SDK consumers calling agent-server API
type ConfigStore interface {
	// GetProviderConfig returns provider configuration and associated API keys.
	GetProviderConfig(providerName llm.ProviderName) (*ProviderConfig, error)

	// GetVirtualKey returns virtual key configuration for access control.
	GetVirtualKey(secretKey string) (*VirtualKeyConfig, error)
}

type LLMGateway struct {
	ConfigStore ConfigStore
}

func NewLLMGateway(ConfigStore ConfigStore) *LLMGateway {
	return &LLMGateway{
		ConfigStore: ConfigStore,
	}
}

func (g *LLMGateway) HandleRequest(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.Response, error) {
	// Construct the provider
	p, err := g.getProvider(ctx, providerName, key)
	if err != nil {
		return nil, err
	}

	// Create the response
	resp := &llm.Response{}

	switch {
	case r.OfResponsesInput != nil:
		respOut, err := g.handleResponsesRequest(ctx, providerName, p, r.OfResponsesInput)
		if err != nil {
			return nil, err
		}

		resp.OfResponsesOutput = respOut
	}

	return resp, nil
}

func (g *LLMGateway) HandleStreamingRequest(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.StreamingResponse, error) {
	// Construct the provider
	p, err := g.getProvider(ctx, providerName, key)
	if err != nil {
		return nil, err
	}

	// Create the response
	resp := &llm.StreamingResponse{}

	switch {
	case r.OfResponsesInput != nil:
		respOut, err := g.handleStreamingResponsesRequest(ctx, providerName, p, r.OfResponsesInput)
		if err != nil {
			return nil, err
		}

		resp.ResponsesStreamData = respOut
		return resp, nil
	}

	return nil, errors.New("invalid request")
}
