package gateway

import (
	"context"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
)

// LLMGatewayAdapter is the interface for making LLM calls.
// Similar to ConversationPersistenceAdapter, it can be implemented by:
// - InternalLLMProvider: uses the internal gateway (for server-side)
// - ExternalLLMProvider: calls agent-server via HTTP (for SDK consumers)
type LLMGatewayAdapter interface {
	// NewResponses makes a non-streaming LLM call
	NewResponses(ctx context.Context, provider llm.ProviderName, req *responses.Request) (*responses.Response, error)

	// NewStreamingResponses makes a streaming LLM call
	NewStreamingResponses(ctx context.Context, provider llm.ProviderName, req *responses.Request) (chan *responses.ResponseChunk, error)
}

// LLMClient wraps an LLMGatewayAdapter and provides a high-level interface
type LLMClient struct {
	LLMGatewayAdapter

	provider llm.ProviderName
	model    string
}

// NewLLMClient creates a new LLM client with the given provider.
func NewLLMClient(p LLMGatewayAdapter, providerName llm.ProviderName, model string) *LLMClient {
	return &LLMClient{
		LLMGatewayAdapter: p,
		provider:          providerName,
		model:             model,
	}
}

func (c *LLMClient) NewResponses(ctx context.Context, in *responses.Request) (*responses.Response, error) {
	in.Model = c.model
	in.Stream = utils.Ptr(false)
	in.Store = utils.Ptr(false)
	return c.LLMGatewayAdapter.NewResponses(ctx, c.provider, in)
}

// NewStreamingResponses invokes the LLM and streams responses via callback
func (c *LLMClient) NewStreamingResponses(ctx context.Context, in *responses.Request) (chan *responses.ResponseChunk, error) {
	in.Model = c.model
	in.Stream = utils.Ptr(true)
	in.Store = utils.Ptr(false)
	return c.LLMGatewayAdapter.NewStreamingResponses(ctx, c.provider, in)
}
