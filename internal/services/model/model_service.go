package model

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ModelService handles business logic for models
type ModelService struct {
	repo *ModelRepo
}

// NewModelService creates a new model service
func NewModelService(repo *ModelRepo) *ModelService {
	return &ModelService{repo: repo}
}

// Create creates a new model
func (s *ModelService) Create(ctx context.Context, projectID uuid.UUID, req *CreateModelRequest) (*Model, error) {
	// Validate that provider belongs to the project
	// This check should ideally be done by checking the provider exists in the project
	// For now, we'll rely on the foreign key constraint

	// Check if model with same name already exists in this project
	existing, err := s.repo.GetByName(ctx, projectID, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("model with name '%s' already exists", req.Name)
	}

	// Ensure parameters is initialized
	if req.Parameters == nil {
		req.Parameters = make(ModelParameters)
	}

	// Create the model
	m, err := s.repo.Create(ctx, projectID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	return m, nil
}

// GetByID retrieves a model by ID
func (s *ModelService) GetByID(ctx context.Context, projectID, id uuid.UUID) (*Model, error) {
	m, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return m, nil
}

// GetByName retrieves a model by name
func (s *ModelService) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*Model, error) {
	m, err := s.repo.GetByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return m, nil
}

// List retrieves all models with optional filtering
func (s *ModelService) List(ctx context.Context, projectID uuid.UUID, providerType *string) ([]*ModelWithProvider, error) {
	models, err := s.repo.List(ctx, projectID, providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	return models, nil
}

// Update updates a model
func (s *ModelService) Update(ctx context.Context, projectID, id uuid.UUID, req *UpdateModelRequest) (*Model, error) {
	// Check if model exists
	existing, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing model: %w", err)
	}

	// If updating name, check if new name already exists
	if req.Name != nil && *req.Name != existing.Name {
		existingByName, err := s.repo.GetByName(ctx, projectID, *req.Name)
		if err == nil && existingByName != nil {
			return nil, fmt.Errorf("model with name '%s' already exists", *req.Name)
		}
	}

	// If parameters is provided but nil, initialize it as empty map
	// Otherwise, merge with existing parameters if partial update is desired
	if req.Parameters != nil && len(*req.Parameters) == 0 {
		// If empty map is provided, use it (replaces all parameters)
		// For partial update, we could merge with existing.Parameters
		// For now, we'll replace entirely when Parameters is provided
	}

	// Update the model
	m, err := s.repo.Update(ctx, projectID, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	return m, nil
}

// Delete deletes a model
func (s *ModelService) Delete(ctx context.Context, projectID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, projectID, id)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}
