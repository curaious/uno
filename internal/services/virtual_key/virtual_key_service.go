package virtual_key

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// VirtualKeyService handles business logic for virtual keys
type VirtualKeyService struct {
	repo *VirtualKeyRepo
}

// NewVirtualKeyService creates a new virtual key service
func NewVirtualKeyService(repo *VirtualKeyRepo) *VirtualKeyService {
	return &VirtualKeyService{repo: repo}
}

// Create creates a new virtual key
func (s *VirtualKeyService) Create(ctx context.Context, req *CreateVirtualKeyRequest) (*VirtualKey, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Check if virtual key with same name already exists
	existing, err := s.repo.GetByName(ctx, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("virtual key with name '%s' already exists", req.Name)
	}

	// Validate providers
	for _, providerType := range req.Providers {
		if !providerType.IsValid() {
			return nil, fmt.Errorf("invalid provider type: %s", providerType)
		}
	}

	// Create the virtual key
	vk, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual key: %w", err)
	}

	return vk, nil
}

// GetByID retrieves a virtual key by ID
func (s *VirtualKeyService) GetByID(ctx context.Context, id uuid.UUID) (*VirtualKey, error) {
	vk, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}

	return vk, nil
}

// GetByName retrieves a virtual key by name
func (s *VirtualKeyService) GetByName(ctx context.Context, name string) (*VirtualKey, error) {
	vk, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}

	return vk, nil
}

// GetBySecretKey retrieves a virtual key by its secret key
func (s *VirtualKeyService) GetBySecretKey(ctx context.Context, secretKey string) (*VirtualKey, error) {
	vk, err := s.repo.GetBySecretKey(ctx, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}

	return vk, nil
}

// List retrieves all virtual keys
func (s *VirtualKeyService) List(ctx context.Context) ([]*VirtualKey, error) {
	virtualKeys, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual keys: %w", err)
	}

	return virtualKeys, nil
}

// Update updates a virtual key
func (s *VirtualKeyService) Update(ctx context.Context, id uuid.UUID, req *UpdateVirtualKeyRequest) (*VirtualKey, error) {
	// Check if virtual key exists
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing virtual key: %w", err)
	}

	// If updating name, check if new name already exists
	if req.Name != nil && *req.Name != existing.Name {
		existingByName, err := s.repo.GetByName(ctx, *req.Name)
		if err == nil && existingByName != nil {
			return nil, fmt.Errorf("virtual key with name '%s' already exists", *req.Name)
		}
	}

	// Validate providers if provided
	if req.Providers != nil {
		for _, providerType := range *req.Providers {
			if !providerType.IsValid() {
				return nil, fmt.Errorf("invalid provider type: %s", providerType)
			}
		}
	}

	// Update the virtual key
	vk, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update virtual key: %w", err)
	}

	return vk, nil
}

// Delete deletes a virtual key
func (s *VirtualKeyService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete virtual key: %w", err)
	}

	return nil
}
