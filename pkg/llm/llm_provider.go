package llm

import (
	"context"
	"slices"

	"github.com/praveen001/uno/pkg/llm/responses"
)

type Provider interface {
	NewResponses(ctx context.Context, in *responses.Request) (*responses.Response, error)
	NewStreamingResponses(ctx context.Context, in *responses.Request) (chan *responses.ResponseChunk, error)
}

type ProviderName string

var (
	ProviderNameOpenAI    ProviderName = "OpenAI"
	ProviderNameAnthropic ProviderName = "Anthropic"
	ProviderNameGemini    ProviderName = "Gemini"
	ProviderNameXAI       ProviderName = "xAI"
)

func GetAllProviderNames() []ProviderName {
	return []ProviderName{
		ProviderNameOpenAI,
		ProviderNameAnthropic,
		ProviderNameGemini,
		ProviderNameXAI,
	}
}

func (p *ProviderName) IsValid() bool {
	return slices.Contains(GetAllProviderNames(), *p)
}
