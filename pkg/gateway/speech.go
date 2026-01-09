package gateway

import (
	"context"

	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/speech"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (g *LLMGateway) handleSpeechRequest(ctx context.Context, providerName llm.ProviderName, p llm.Provider, in *speech.Request) (*speech.Response, error) {
	ctx, span := tracer.Start(ctx, "LLM.Speech")
	defer span.End()

	span.SetAttributes(
		attribute.String("llm.provider", string(providerName)),
		attribute.String("llm.model", in.Model),
		attribute.String("llm.request_type", "Speech"),
	)

	out, err := p.NewSpeech(ctx, in)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Add output attributes
	//if out.Usage.TotalTokens > 0 {
	//	span.SetAttributes(
	//		attribute.Int64("llm.usage.prompt_tokens", out.Usage.PromptTokens),
	//		attribute.Int64("llm.usage.completion_tokens", out.Usage.CompletionTokens),
	//		attribute.Int64("llm.usage.total_tokens", out.Usage.TotalTokens),
	//	)
	//}

	return out, nil
}
