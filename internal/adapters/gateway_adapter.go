package adapters

import (
	"context"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
)

// InternalLLMGateway uses the internal LLMGatewayAdapter for server-side use.
// This is used within the agent-server where we have direct access to services.
// It handles virtual key resolution and provider configuration from the database.
type InternalLLMGateway struct {
	gateway *gateway.LLMGateway
	key     string // Virtual key or direct API key
}

// NewInternalLLMGateway creates a provider using the internal gateway.
// The key can be a virtual key (sk-amg-xxx) which will be resolved to actual API keys,
// or a direct API key for the provider.
func NewInternalLLMGateway(gw *gateway.LLMGateway, key string) *InternalLLMGateway {
	return &InternalLLMGateway{
		gateway: gw,
		key:     key,
	}
}

func (p *InternalLLMGateway) NewResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (*responses.Response, error) {
	llmReq := &llm.Request{
		OfResponsesInput: req,
	}

	resp, err := p.gateway.HandleRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return resp.OfResponsesOutput, nil
}

func (p *InternalLLMGateway) NewStreamingResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (chan *responses.ResponseChunk, error) {
	llmReq := &llm.Request{
		OfResponsesInput: req,
	}

	streamResp, err := p.gateway.HandleStreamingRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return streamResp.ResponsesStreamData, nil
}
