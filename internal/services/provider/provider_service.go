package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/praveen001/uno/pkg/llm"
)

// ProviderService handles business logic for API keys
type ProviderService struct {
	repo *ProviderRepo
}

// NewProviderService creates a new provider service
func NewProviderService(repo *ProviderRepo) *ProviderService {
	return &ProviderService{repo: repo}
}

// Create creates a new API key
func (s *ProviderService) Create(ctx context.Context, req *CreateAPIKeyRequest) (*APIKey, error) {
	// Validate provider type
	if !req.ProviderType.IsValid() {
		return nil, fmt.Errorf("invalid provider type: %s", req.ProviderType)
	}

	// Validate name
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Validate API key
	if req.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Check if API key with same name already exists for this provider type
	existing, err := s.repo.GetByName(ctx, req.ProviderType, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("API key with name '%s' already exists for provider type '%s'", req.Name, req.ProviderType)
	}

	// Create the API key
	apiKey, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKey, nil
}

// GetByName retrieves an API key by provider type and name
func (s *ProviderService) GetByName(ctx context.Context, providerType llm.ProviderName, name string) (*APIKey, error) {
	apiKey, err := s.repo.GetByName(ctx, providerType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return apiKey, nil
}

// List retrieves all API keys with optional filtering
func (s *ProviderService) List(ctx context.Context, providerType *llm.ProviderName, enabledOnly bool) ([]*APIKey, error) {
	apiKeys, err := s.repo.List(ctx, providerType, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return apiKeys, nil
}

// GetDefaultAPIKey retrieves the default API key for a provider type
func (s *ProviderService) GetDefaultAPIKey(ctx context.Context, providerType llm.ProviderName) (*APIKey, error) {
	apiKey, err := s.repo.GetDefaultAPIKey(ctx, providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get default API key: %w", err)
	}

	return apiKey, nil
}

// Update updates an API key
func (s *ProviderService) Update(ctx context.Context, id uuid.UUID, req *UpdateAPIKeyRequest) (*APIKey, error) {
	// Check if API key exists
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing API key: %w", err)
	}

	// If updating name, check if new name already exists for this provider type
	if req.Name != nil && *req.Name != existing.Name {
		existingByName, err := s.repo.GetByName(ctx, existing.ProviderType, *req.Name)
		if err == nil && existingByName != nil {
			return nil, fmt.Errorf("API key with name '%s' already exists for provider type '%s'", *req.Name, existing.ProviderType)
		}
	}

	// Validate API key if provided
	if req.APIKey != nil && *req.APIKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	// Update the API key
	apiKey, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	return apiKey, nil
}

// Delete deletes an API key
func (s *ProviderService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// GetProviderConfig retrieves provider config by provider type
func (s *ProviderService) GetProviderConfig(ctx context.Context, providerType llm.ProviderName) (*ProviderConfig, error) {
	if !providerType.IsValid() {
		return nil, fmt.Errorf("invalid provider type: %s", providerType)
	}

	config, err := s.repo.GetProviderConfig(ctx, providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider config: %w", err)
	}

	return config, nil
}

// CreateOrUpdateProviderConfig creates or updates provider config
func (s *ProviderService) CreateOrUpdateProviderConfig(ctx context.Context, req *CreateProviderConfigRequest) (*ProviderConfig, error) {
	if !req.ProviderType.IsValid() {
		return nil, fmt.Errorf("invalid provider type: %s", req.ProviderType)
	}

	config, err := s.repo.CreateOrUpdateProviderConfig(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update provider config: %w", err)
	}

	return config, nil
}

// UpdateProviderConfig updates provider config
func (s *ProviderService) UpdateProviderConfig(ctx context.Context, providerType llm.ProviderName, req *UpdateProviderConfigRequest) (*ProviderConfig, error) {
	if !providerType.IsValid() {
		return nil, fmt.Errorf("invalid provider type: %s", providerType)
	}

	config, err := s.repo.UpdateProviderConfig(ctx, providerType, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider config: %w", err)
	}

	return config, nil
}

// ListProviderConfigs retrieves all provider configs
func (s *ProviderService) ListProviderConfigs(ctx context.Context) ([]*ProviderConfig, error) {
	configs, err := s.repo.ListProviderConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider configs: %w", err)
	}

	return configs, nil
}
