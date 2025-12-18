package client

import (
	"context"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
)

// LLMGateway is the interface for making LLM calls.
// Similar to ConversationPersistenceManager, it can be implemented by:
// - InternalLLMProvider: uses the internal gateway (for server-side)
// - ExternalLLMProvider: calls agent-server via HTTP (for SDK consumers)
type LLMGateway interface {
	// NewResponses makes a non-streaming LLM call
	NewResponses(ctx context.Context, provider llm.ProviderName, req *responses.Request) (*responses.Response, error)

	// NewStreamingResponses makes a streaming LLM call
	NewStreamingResponses(ctx context.Context, provider llm.ProviderName, req *responses.Request) (chan *responses.ResponseChunk, error)
}

// LLMClient wraps an LLMGateway and provides a high-level interface
type LLMClient struct {
	LLMGateway

	provider llm.ProviderName
	model    string
	tools    []core.Tool
	output   any
}

// NewLLMClient creates a new LLM client with the given provider.
func NewLLMClient(p LLMGateway, providerName llm.ProviderName, model string) *LLMClient {
	return &LLMClient{
		LLMGateway: p,
		provider:   providerName,
		model:      model,
	}
}

func (c *LLMClient) NewResponses(ctx context.Context, in *responses.Request) (*responses.Response, error) {
	in.Model = c.model
	in.Stream = utils.Ptr(false)
	in.Store = utils.Ptr(false)
	return c.LLMGateway.NewResponses(ctx, c.provider, in)
}

// NewStreamingResponses invokes the LLM and streams responses via callback
func (c *LLMClient) NewStreamingResponses(ctx context.Context, in *responses.Request) (chan *responses.ResponseChunk, error) {
	in.Model = c.model
	in.Stream = utils.Ptr(true)
	in.Store = utils.Ptr(false)
	return c.LLMGateway.NewStreamingResponses(ctx, c.provider, in)
}
