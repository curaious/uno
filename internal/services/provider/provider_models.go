package provider

import (
	"github.com/curaious/uno/pkg/llm"
)

// ProviderModels contains the available models for each provider
var ProviderModels = map[llm.ProviderName][]string{
	llm.ProviderNameOpenAI: {
		"gpt-5",
		"gpt-5-mini",
		"gpt-5-nano",
		"gpt-5.1",
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"gpt-4o",
		"gpt-4o-mini",
		"o3",
		"o3-mini",
		"o4-mini",
	},
	llm.ProviderNameAnthropic: {
		"claude-haiku-4-5",
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-opus-4-1-20250805",
		"claude-sonnet-4-0",
		"claude-3-7-sonnet",
		"claude-3-5-sonnet",
		"claude-3-5-haiku",
	},
	llm.ProviderNameGemini: {
		"gemini-3-flash-preview",
		"gemini-3-pro-preview",
		"gemini-3-pro",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.0-flash",
		"gemini-2.5-flash-image",
		"gemini-1.5-pro",
	},
	llm.ProviderNameXAI: {
		"grok-4",
		"grok-4-heavy",
		"grok-4.1-fast-reasoning",
		"grok-4.1-fast-non-reasoning",
		"grok-4-fast-reasoning",
		"grok-4-fast-non-reasoning",
		"grok-3",
		"grok-3-mini",
		"grok-2.5",
	},
	llm.ProviderNameOllama: {
		"llama3.1:latest",
		"mistral:latest",
	},
}

// ProviderModelsResponse represents the response structure for provider models API
type ProviderModelsResponse struct {
	Providers map[string]ProviderModelsData `json:"providers"`
}

// ProviderModelsData represents the models data for a provider
type ProviderModelsData struct {
	Models []string `json:"models"`
}

// GetProviderModelsResponse returns the provider models in the API response format
func GetProviderModelsResponse() ProviderModelsResponse {
	response := ProviderModelsResponse{
		Providers: make(map[string]ProviderModelsData),
	}

	for providerType, models := range ProviderModels {
		response.Providers[string(providerType)] = ProviderModelsData{
			Models: models,
		}
	}

	return response
}
