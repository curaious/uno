package sdk

import (
	"fmt"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
)

// InMemoryConfigStore implements gateway.ConfigStore for SDK use.
// It holds API keys and provider configs in memory.
type InMemoryConfigStore struct {
	providerConfigs map[llm.ProviderName]*gateway.ProviderConfig
}

// NewInMemoryConfigStore creates a config store with full provider options.
func NewInMemoryConfigStore(configs []*gateway.ProviderConfig) *InMemoryConfigStore {
	store := &InMemoryConfigStore{
		providerConfigs: make(map[llm.ProviderName]*gateway.ProviderConfig),
	}

	for _, config := range configs {
		// Set provider config
		store.providerConfigs[config.ProviderName] = &gateway.ProviderConfig{
			ProviderName:  config.ProviderName,
			BaseURL:       config.BaseURL,
			CustomHeaders: config.CustomHeaders,
			ApiKeys:       config.ApiKeys,
		}
	}

	return store
}

func (s *InMemoryConfigStore) GetProviderConfig(providerName llm.ProviderName) (*gateway.ProviderConfig, error) {
	config := s.providerConfigs[providerName]

	if len(config.ApiKeys) == 0 {
		return nil, fmt.Errorf("no API key configured for provider %s", providerName)
	}

	return config, nil
}

func (s *InMemoryConfigStore) GetVirtualKey(secretKey string) (*gateway.VirtualKeyConfig, error) {
	// In-memory store doesn't support virtual keys - they're managed by agent-server
	return nil, fmt.Errorf("virtual keys are not supported in direct mode")
}
