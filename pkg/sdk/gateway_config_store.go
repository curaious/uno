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
	apiKeys         map[llm.ProviderName][]*gateway.APIKeyConfig
}

// ProviderOptions configures a provider for in-memory use
type ProviderConfig struct {
	ProviderName  llm.ProviderName
	BaseURL       string
	CustomHeaders map[string]string
	Keys          []*ProviderKey
}

type ProviderKey struct {
	Name string
	Key  string
}

// NewInMemoryConfigStore creates a config store with full provider options.
func NewInMemoryConfigStore(configs []*ProviderConfig) *InMemoryConfigStore {
	store := &InMemoryConfigStore{
		providerConfigs: make(map[llm.ProviderName]*gateway.ProviderConfig),
		apiKeys:         make(map[llm.ProviderName][]*gateway.APIKeyConfig),
	}

	for _, config := range configs {
		// Set provider config
		store.providerConfigs[config.ProviderName] = &gateway.ProviderConfig{
			ProviderName:  config.ProviderName,
			BaseURL:       config.BaseURL,
			CustomHeaders: config.CustomHeaders,
		}

		var keys []*gateway.APIKeyConfig
		for _, key := range config.Keys {
			keys = append(keys, &gateway.APIKeyConfig{
				ProviderName: config.ProviderName,
				APIKey:       key.Key,
				Name:         key.Name,
				Enabled:      true,
				IsDefault:    true,
			})
		}
		store.apiKeys[config.ProviderName] = keys
	}

	return store
}

func (s *InMemoryConfigStore) GetProviderConfig(providerName llm.ProviderName) (*gateway.ProviderConfig, []*gateway.APIKeyConfig, error) {
	keys := s.apiKeys[providerName]
	if len(keys) == 0 {
		return nil, nil, fmt.Errorf("no API key configured for provider %s", providerName)
	}

	config := s.providerConfigs[providerName]
	return config, keys, nil
}

func (s *InMemoryConfigStore) GetVirtualKey(secretKey string) (*gateway.VirtualKeyConfig, error) {
	// In-memory store doesn't support virtual keys - they're managed by agent-server
	return nil, fmt.Errorf("virtual keys are not supported in direct mode")
}
