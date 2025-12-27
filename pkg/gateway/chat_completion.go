package gateway

import (
	"context"

	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/chat_completion"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (g *LLMGateway) handleChatCompletionRequest(ctx context.Context, providerName llm.ProviderName, p llm.Provider, in *chat_completion.Request) (*chat_completion.Response, error) {
	ctx, span := tracer.Start(ctx, "LLM.ChatCompletion")
	defer span.End()

	span.SetAttributes(
		attribute.String("llm.provider", string(providerName)),
		attribute.String("llm.model", in.Model),
		attribute.String("llm.request_type", "ChatCompletion"),
	)

	out, err := p.NewChatCompletion(ctx, in)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Add output attributes
	if out.Usage.TotalTokens > 0 {
		span.SetAttributes(
			attribute.Int64("llm.usage.prompt_tokens", out.Usage.PromptTokens),
			attribute.Int64("llm.usage.completion_tokens", out.Usage.CompletionTokens),
			attribute.Int64("llm.usage.total_tokens", out.Usage.TotalTokens),
		)
	}

	return out, nil
}
