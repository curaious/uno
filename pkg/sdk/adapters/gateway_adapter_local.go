package adapters

import (
	"context"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
)

// LocalLLMGateway uses the internal LLMGatewayAdapter for server-side use.
// This is used within the agent-server where we have direct access to services.
// It handles virtual key resolution and provider configuration from the database.
type LocalLLMGateway struct {
	gateway *gateway.LLMGateway
}

// NewLocalLLMGateway creates a provider using the internal gateway.
// The key can be a virtual key (sk-amg-xxx) which will be resolved to actual API keys,
// or a direct API key for the provider.
func NewLocalLLMGateway(gw *gateway.LLMGateway) *LocalLLMGateway {
	return &LocalLLMGateway{
		gateway: gw,
	}
}

func (p *LocalLLMGateway) NewResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (*responses.Response, error) {
	llmReq := &llm.Request{
		OfResponsesInput: req,
	}

	resp, err := p.gateway.HandleRequest(ctx, providerName, p.getKey(providerName), llmReq)
	if err != nil {
		return nil, err
	}

	return resp.OfResponsesOutput, nil
}

func (p *LocalLLMGateway) NewStreamingResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (chan *responses.ResponseChunk, error) {
	llmReq := &llm.Request{
		OfResponsesInput: req,
	}

	streamResp, err := p.gateway.HandleStreamingRequest(ctx, providerName, p.getKey(providerName), llmReq)
	if err != nil {
		return nil, err
	}

	return streamResp.ResponsesStreamData, nil
}

func (p *LocalLLMGateway) getKey(providerName llm.ProviderName) string {
	_, keys, err := p.gateway.ConfigStore.GetProviderConfig(providerName)
	if err != nil {
		return ""
	}

	if len(keys) == 0 {
		return ""
	}

	return keys[0].APIKey
}
