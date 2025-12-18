package adapters

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/praveen001/uno/internal/services/provider"
	"github.com/praveen001/uno/internal/services/virtual_key"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
)

// ServiceConfigStore implements gateway.ConfigStore using provider and virtual key services.
// This is used server-side where we have access to the database services.
type ServiceConfigStore struct {
	providerService   *provider.ProviderService
	virtualKeyService *virtual_key.VirtualKeyService
}

// NewServiceConfigStore creates a config store backed by database services.
func NewServiceConfigStore(providerSvc *provider.ProviderService, virtualKeySvc *virtual_key.VirtualKeyService) *ServiceConfigStore {
	return &ServiceConfigStore{
		providerService:   providerSvc,
		virtualKeyService: virtualKeySvc,
	}
}

func (s *ServiceConfigStore) GetProviderConfig(providerName llm.ProviderName) (*gateway.ProviderConfig, []*gateway.APIKeyConfig, error) {
	ctx := context.Background()

	// Get provider config from service
	svcConfig, err := s.providerService.GetProviderConfig(ctx, providerName)
	if err != nil {
		return nil, nil, err
	}

	// Get API keys from service
	svcKeys, err := s.providerService.List(ctx, &providerName, true)
	if err != nil {
		return nil, nil, err
	}

	if len(svcKeys) == 0 {
		return nil, nil, fmt.Errorf("no enabled api keys found for provider %s", providerName)
	}

	// Process environment variable templates in API keys
	envData := map[string]string{}
	for _, env := range os.Environ() {
		frag := strings.Split(env, "=")
		if len(frag) >= 2 {
			envData[frag[0]] = strings.Join(frag[1:], "=")
		}
	}

	// Convert to gateway types
	var gwConfig *gateway.ProviderConfig
	if svcConfig != nil {
		gwConfig = &gateway.ProviderConfig{
			ProviderName: svcConfig.ProviderType,
		}
		if svcConfig.BaseURL != nil {
			gwConfig.BaseURL = *svcConfig.BaseURL
		}
		if svcConfig.CustomHeaders != nil {
			gwConfig.CustomHeaders = svcConfig.CustomHeaders
		}
	}

	gwKeys := make([]*gateway.APIKeyConfig, 0, len(svcKeys))
	for _, svcKey := range svcKeys {
		apiKey := utils.TryAndParseAsTemplate(svcKey.APIKey, map[string]any{
			"Env": envData,
		})
		gwKeys = append(gwKeys, &gateway.APIKeyConfig{
			ProviderName: svcKey.ProviderType,
			APIKey:       apiKey,
			Name:         svcKey.Name,
			Enabled:      svcKey.Enabled,
			IsDefault:    svcKey.IsDefault,
		})
	}

	return gwConfig, gwKeys, nil
}

func (s *ServiceConfigStore) GetVirtualKey(secretKey string) (*gateway.VirtualKeyConfig, error) {
	ctx := context.Background()

	if secretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}

	// Get virtual key from service
	svcKey, err := s.virtualKeyService.GetBySecretKey(ctx, secretKey)
	if err != nil {
		return nil, err
	}

	// Convert to gateway type
	return &gateway.VirtualKeyConfig{
		SecretKey:        svcKey.SecretKey,
		AllowedProviders: svcKey.Providers,
		AllowedModels:    svcKey.ModelNames,
	}, nil
}
