package adapters

import (
	"context"

	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/chat_completion"
	"github.com/curaious/uno/pkg/llm/embeddings"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/curaious/uno/pkg/llm/speech"
)

// InternalLLMGateway uses the internal LLMGatewayAdapter for server-side use.
// This is used within the agent-server where we have direct access to services.
// It handles virtual key resolution and provider configuration from the database.
type InternalLLMGateway struct {
	gateway *gateway.LLMGateway
	key     string // Virtual key or direct API key
}

// NewInternalLLMGateway creates a provider using the internal gateway.
// The key can be a virtual key (sk-uno-xxx) which will be resolved to actual API keys,
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

func (p *InternalLLMGateway) NewEmbedding(ctx context.Context, providerName llm.ProviderName, req *embeddings.Request) (*embeddings.Response, error) {
	llmReq := &llm.Request{
		OfEmbeddingsInput: req,
	}

	resp, err := p.gateway.HandleRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return resp.OfEmbeddingsOutput, nil
}

func (p *InternalLLMGateway) NewChatCompletion(ctx context.Context, providerName llm.ProviderName, req *chat_completion.Request) (*chat_completion.Response, error) {
	llmReq := &llm.Request{
		OfChatCompletionInput: req,
	}

	resp, err := p.gateway.HandleRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return resp.OfChatCompletionOutput, nil
}

func (p *InternalLLMGateway) NewStreamingChatCompletion(ctx context.Context, providerName llm.ProviderName, req *chat_completion.Request) (chan *chat_completion.ResponseChunk, error) {
	llmReq := &llm.Request{
		OfChatCompletionInput: req,
	}

	resp, err := p.gateway.HandleStreamingRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return resp.ChatCompletionStreamData, nil
}

func (p *InternalLLMGateway) NewSpeech(ctx context.Context, providerName llm.ProviderName, req *speech.Request) (*speech.Response, error) {
	llmReq := &llm.Request{
		OfSpeech: req,
	}

	resp, err := p.gateway.HandleRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return resp.OfSpeech, nil
}

func (p *InternalLLMGateway) NewStreamingSpeech(ctx context.Context, providerName llm.ProviderName, req *speech.Request) (chan *speech.ResponseChunk, error) {
	llmReq := &llm.Request{
		OfSpeech: req,
	}

	resp, err := p.gateway.HandleStreamingRequest(ctx, providerName, p.key, llmReq)
	if err != nil {
		return nil, err
	}

	return resp.SpeechStreamData, nil
}
