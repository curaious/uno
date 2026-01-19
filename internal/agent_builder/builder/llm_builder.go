package builder

import (
	"github.com/curaious/uno/internal/adapters"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
)

func BuildLLMClient(llmGateway *gateway.LLMGateway, virtualKey string, providerName llm.ProviderName, modelID string) llm.Provider {
	return gateway.NewLLMClient(
		adapters.NewInternalLLMGateway(llmGateway, virtualKey),
		providerName,
		modelID,
	)
}
