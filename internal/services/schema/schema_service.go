package schema

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// SchemaService handles business logic for schemas
type SchemaService struct {
	repo *SchemaRepo
}

// NewSchemaService creates a new schema service
func NewSchemaService(repo *SchemaRepo) *SchemaService {
	return &SchemaService{repo: repo}
}

// CreateSchema creates a new schema
func (s *SchemaService) CreateSchema(ctx context.Context, projectID uuid.UUID, req *CreateSchemaRequest) (*Schema, error) {
	// Check if schema with same name already exists
	existing, err := s.repo.GetByName(ctx, projectID, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("schema with name '%s' already exists", req.Name)
	}

	// Validate the JSON schema
	if err := ValidateJSONSchema(req.Schema); err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}

	// Set default source type if not provided
	if req.SourceType == "" {
		req.SourceType = SourceTypeManual
	}

	schema, err := s.repo.Create(ctx, projectID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return schema, nil
}

// GetSchema retrieves a schema by ID
func (s *SchemaService) GetSchema(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (*Schema, error) {
	schema, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return schema, nil
}

// GetSchemaByName retrieves a schema by name
func (s *SchemaService) GetSchemaByName(ctx context.Context, projectID uuid.UUID, name string) (*Schema, error) {
	schema, err := s.repo.GetByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return schema, nil
}

// ListSchemas retrieves all schemas for a project
func (s *SchemaService) ListSchemas(ctx context.Context, projectID uuid.UUID) ([]*Schema, error) {
	schemas, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	return schemas, nil
}

// UpdateSchema updates a schema
func (s *SchemaService) UpdateSchema(ctx context.Context, projectID uuid.UUID, id uuid.UUID, req *UpdateSchemaRequest) (*Schema, error) {
	// If name is being updated, check for duplicates
	if req.Name != nil {
		existing, err := s.repo.GetByName(ctx, projectID, *req.Name)
		if err == nil && existing != nil && existing.ID != id {
			return nil, fmt.Errorf("schema with name '%s' already exists", *req.Name)
		}
	}

	// Validate the JSON schema if it's being updated
	if req.Schema != nil {
		if err := ValidateJSONSchema(req.Schema); err != nil {
			return nil, fmt.Errorf("invalid JSON schema: %w", err)
		}
	}

	schema, err := s.repo.Update(ctx, projectID, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update schema: %w", err)
	}

	return schema, nil
}

// DeleteSchema deletes a schema
func (s *SchemaService) DeleteSchema(ctx context.Context, projectID uuid.UUID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, projectID, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	return nil
}
