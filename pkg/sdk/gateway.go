package sdk

import (
	internal_adapters "github.com/curaious/uno/internal/adapters"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/sdk/adapters"
)

type LLMOptions struct {
	Provider llm.ProviderName
	Model    string
}

// NewLLM creates a new LLMClient that provides access to multiple LLM providers.
func (c *SDK) NewLLM(opts LLMOptions) llm.Provider {
	return gateway.NewLLMClient(
		c.getGatewayAdapter(opts.Provider),
		opts.Provider,
		opts.Model,
	)
}

func (c *SDK) getGatewayAdapter(providerName llm.ProviderName) gateway.LLMGatewayAdapter {
	if c.directMode {
		return internal_adapters.NewInternalLLMGateway(gateway.NewLLMGateway(c.llmConfigs), getKey(c.llmConfigs, providerName))
	}

	return adapters.NewExternalLLMGateway(c.endpoint, c.virtualKey)
}

func getKey(cfgStore gateway.ConfigStore, providerName llm.ProviderName) string {
	providerConfig, err := cfgStore.GetProviderConfig(providerName)
	if err != nil {
		return ""
	}

	if len(providerConfig.ApiKeys) == 0 {
		return ""
	}

	if len(providerConfig.ApiKeys) == 1 {
		return providerConfig.ApiKeys[0].APIKey
	}

	// Weight random selection
	weights := make([]int, len(providerConfig.ApiKeys))
	for idx, key := range providerConfig.ApiKeys {
		weights[idx] = key.Weight
	}

	return providerConfig.ApiKeys[utils.WeightedRandomIndex(weights)].APIKey
}
