package adapters

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/curaious/uno/internal/pubsub"
	"github.com/curaious/uno/internal/services/provider"
	"github.com/curaious/uno/internal/services/virtual_key"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
)

// ServiceConfigStore implements gateway.ConfigStore using provider and virtual key services.
// This is used server-side where we have access to the database services.
// It maintains an in-memory cache that is automatically updated via PostgreSQL LISTEN/NOTIFY.
type ServiceConfigStore struct {
	providerConfigs map[llm.ProviderName]*gateway.ProviderConfig
	virtualKeys     map[string]*gateway.VirtualKeyConfig

	providerService   *provider.ProviderService
	virtualKeyService *virtual_key.VirtualKeyService

	mu sync.RWMutex
}

// NewServiceConfigStore creates a config store backed by database services.
func NewServiceConfigStore(providerSvc *provider.ProviderService, virtualKeySvc *virtual_key.VirtualKeyService) *ServiceConfigStore {
	store := &ServiceConfigStore{
		providerConfigs:   make(map[llm.ProviderName]*gateway.ProviderConfig),
		virtualKeys:       make(map[string]*gateway.VirtualKeyConfig),
		providerService:   providerSvc,
		virtualKeyService: virtualKeySvc,
	}

	// Initial load of all configurations
	store.reloadProviderConfigs()
	store.reloadAPIKeys()
	store.reloadVirtualKeys()

	return store
}

// SubscribeToPubSub subscribes to configuration change notifications.
// This should be called after the pubsub is started.
func (s *ServiceConfigStore) SubscribeToPubSub(ps *pubsub.PubSub) {
	ps.Subscribe(func(event pubsub.ConfigChangeEvent) {
		slog.Debug("ServiceConfigStore received config change",
			slog.String("table", string(event.ChangeType)),
			slog.String("operation", event.Operation))

		switch event.ChangeType {
		case pubsub.ChangeTypeProviderConfig:
			s.reloadProviderConfigs()
		case pubsub.ChangeTypeAPIKey:
			s.reloadAPIKeys()
		case pubsub.ChangeTypeVirtualKey, pubsub.ChangeTypeVirtualKeyProvider, pubsub.ChangeTypeVirtualKeyModel:
			s.reloadVirtualKeys()
		}
	})
}

// reloadProviderConfigs reloads all provider configurations from the database
func (s *ServiceConfigStore) reloadProviderConfigs() {
	ctx := context.Background()

	providerConfigs, err := s.providerService.ListProviderConfigs(ctx)
	if err != nil {
		slog.Error("Failed to reload provider configs", slog.Any("error", err))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update provider configs (base_url, custom_headers) while preserving API keys
	for _, providerConfig := range providerConfigs {
		existing, exists := s.providerConfigs[providerConfig.ProviderType]
		if !exists {
			existing = &gateway.ProviderConfig{
				ProviderName: providerConfig.ProviderType,
			}
			s.providerConfigs[providerConfig.ProviderType] = existing
		}

		if providerConfig.BaseURL != nil {
			existing.BaseURL = *providerConfig.BaseURL
		} else {
			existing.BaseURL = ""
		}
		if providerConfig.CustomHeaders != nil {
			existing.CustomHeaders = providerConfig.CustomHeaders
		} else {
			existing.CustomHeaders = nil
		}
	}

	slog.Debug("Reloaded provider configs", slog.Int("count", len(providerConfigs)))
}

// reloadAPIKeys reloads all API keys from the database
func (s *ServiceConfigStore) reloadAPIKeys() {
	ctx := context.Background()

	providerKeys, err := s.providerService.List(ctx, nil, false)
	if err != nil {
		slog.Error("Failed to reload API keys", slog.Any("error", err))
		return
	}

	// Process environment variable templates in API keys
	envData := getEnvData()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing API keys from all providers
	for _, config := range s.providerConfigs {
		config.ApiKeys = nil
	}

	// Repopulate API keys
	for _, providerKey := range providerKeys {
		if _, exists := s.providerConfigs[providerKey.ProviderType]; !exists {
			s.providerConfigs[providerKey.ProviderType] = &gateway.ProviderConfig{
				ProviderName: providerKey.ProviderType,
			}
		}

		apiKey := utils.TryAndParseAsTemplate(providerKey.APIKey, map[string]any{
			"Env": envData,
		})
		s.providerConfigs[providerKey.ProviderType].ApiKeys = append(
			s.providerConfigs[providerKey.ProviderType].ApiKeys,
			&gateway.APIKeyConfig{
				ProviderName: providerKey.ProviderType,
				APIKey:       apiKey,
				Name:         providerKey.Name,
				Enabled:      providerKey.Enabled,
				IsDefault:    providerKey.IsDefault,
			},
		)
	}

	slog.Debug("Reloaded API keys", slog.Int("count", len(providerKeys)))
}

// reloadVirtualKeys reloads all virtual keys from the database
func (s *ServiceConfigStore) reloadVirtualKeys() {
	ctx := context.Background()

	virtualKeys, err := s.virtualKeyService.List(ctx)
	if err != nil {
		slog.Error("Failed to reload virtual keys", slog.Any("error", err))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear and repopulate
	s.virtualKeys = make(map[string]*gateway.VirtualKeyConfig)
	for _, virtualKey := range virtualKeys {
		vk := &gateway.VirtualKeyConfig{
			SecretKey:        virtualKey.SecretKey,
			AllowedProviders: virtualKey.Providers,
			AllowedModels:    virtualKey.ModelNames,
			RateLimits:       []gateway.RateLimit{},
		}

		if virtualKey.RateLimits != nil {
			for _, rateLimit := range virtualKey.RateLimits {
				vk.RateLimits = append(vk.RateLimits, gateway.RateLimit{
					Unit:  rateLimit.Unit,
					Limit: rateLimit.Limit,
				})
			}
		}

		s.virtualKeys[virtualKey.SecretKey] = vk

	}

	slog.Debug("Reloaded virtual keys", slog.Int("count", len(virtualKeys)))
}

// getEnvData returns environment variables as a map
func getEnvData() map[string]string {
	envData := map[string]string{}
	for _, env := range os.Environ() {
		frag := strings.Split(env, "=")
		if len(frag) >= 2 {
			envData[frag[0]] = strings.Join(frag[1:], "=")
		}
	}
	return envData
}

func (s *ServiceConfigStore) GetProviderConfig(providerName llm.ProviderName) (*gateway.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gwConfig := s.providerConfigs[providerName]

	if gwConfig == nil {
		return nil, errors.New("provider not exist")
	}

	return gwConfig, nil
}

func (s *ServiceConfigStore) GetVirtualKey(secretKey string) (*gateway.VirtualKeyConfig, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	vk, ok := s.virtualKeys[secretKey]
	if !ok {
		return nil, fmt.Errorf("secret key not exist")
	}

	return vk, nil
}
