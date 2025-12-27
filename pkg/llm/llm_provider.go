package llm

import (
	"context"
	"slices"

	"github.com/praveen001/uno/pkg/llm/chat_completion"
	"github.com/praveen001/uno/pkg/llm/embeddings"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type Provider interface {
	NewResponses(ctx context.Context, in *responses.Request) (*responses.Response, error)
	NewStreamingResponses(ctx context.Context, in *responses.Request) (chan *responses.ResponseChunk, error)
	NewEmbedding(ctx context.Context, in *embeddings.Request) (*embeddings.Response, error)
	NewChatCompletion(ctx context.Context, in *chat_completion.Request) (*chat_completion.Response, error)
}

type ProviderName string

var (
	ProviderNameOpenAI    ProviderName = "OpenAI"
	ProviderNameAnthropic ProviderName = "Anthropic"
	ProviderNameGemini    ProviderName = "Gemini"
	ProviderNameXAI       ProviderName = "xAI"
	ProviderNameOllama    ProviderName = "Ollama"
)

func GetAllProviderNames() []ProviderName {
	return []ProviderName{
		ProviderNameOpenAI,
		ProviderNameAnthropic,
		ProviderNameGemini,
		ProviderNameXAI,
		ProviderNameOllama,
	}
}

func (p *ProviderName) IsValid() bool {
	return slices.Contains(GetAllProviderNames(), *p)
}
