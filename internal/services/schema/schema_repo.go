package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SchemaRepo handles database operations for schemas
type SchemaRepo struct {
	db *sqlx.DB
}

// NewSchemaRepo creates a new schema repository
func NewSchemaRepo(db *sqlx.DB) *SchemaRepo {
	return &SchemaRepo{db: db}
}

// Create creates a new schema
func (r *SchemaRepo) Create(ctx context.Context, projectID uuid.UUID, req *CreateSchemaRequest) (*Schema, error) {
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = SourceTypeManual
	}

	query := `
		INSERT INTO schemas (project_id, name, description, schema, source_type, source_content)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, project_id, name, description, schema, source_type, source_content, created_at, updated_at
	`

	var schema Schema
	err := r.db.GetContext(ctx, &schema, query, projectID, req.Name, req.Description, req.Schema, sourceType, req.SourceContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &schema, nil
}

// GetByID retrieves a schema by ID
func (r *SchemaRepo) GetByID(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (*Schema, error) {
	query := `
		SELECT id, project_id, name, description, schema, source_type, source_content, created_at, updated_at
		FROM schemas
		WHERE id = $1 AND project_id = $2
	`

	var schema Schema
	err := r.db.GetContext(ctx, &schema, query, id, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schema not found")
		}
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return &schema, nil
}

// GetByName retrieves a schema by name
func (r *SchemaRepo) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*Schema, error) {
	query := `
		SELECT id, project_id, name, description, schema, source_type, source_content, created_at, updated_at
		FROM schemas
		WHERE name = $1 AND project_id = $2
	`

	var schema Schema
	err := r.db.GetContext(ctx, &schema, query, name, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schema not found")
		}
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return &schema, nil
}

// List retrieves all schemas for a project
func (r *SchemaRepo) List(ctx context.Context, projectID uuid.UUID) ([]*Schema, error) {
	query := `
		SELECT id, project_id, name, description, schema, source_type, source_content, created_at, updated_at
		FROM schemas
		WHERE project_id = $1
		ORDER BY created_at DESC
	`

	var schemas []*Schema
	err := r.db.SelectContext(ctx, &schemas, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	return schemas, nil
}

// Update updates a schema
func (r *SchemaRepo) Update(ctx context.Context, projectID uuid.UUID, id uuid.UUID, req *UpdateSchemaRequest) (*Schema, error) {
	// First, get the existing schema
	existing, err := r.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}

	schemaData := existing.Schema
	if req.Schema != nil {
		schemaData = req.Schema
	}

	sourceType := existing.SourceType
	if req.SourceType != nil {
		sourceType = *req.SourceType
	}

	sourceContent := existing.SourceContent
	if req.SourceContent != nil {
		sourceContent = req.SourceContent
	}

	query := `
		UPDATE schemas
		SET name = $1, description = $2, schema = $3, source_type = $4, source_content = $5, updated_at = NOW()
		WHERE id = $6 AND project_id = $7
		RETURNING id, project_id, name, description, schema, source_type, source_content, created_at, updated_at
	`

	var schema Schema
	err = r.db.GetContext(ctx, &schema, query, name, description, schemaData, sourceType, sourceContent, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to update schema: %w", err)
	}

	return &schema, nil
}

// Delete deletes a schema
func (r *SchemaRepo) Delete(ctx context.Context, projectID uuid.UUID, id uuid.UUID) error {
	query := `DELETE FROM schemas WHERE id = $1 AND project_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schema not found")
	}

	return nil
}

// ValidateJSONSchema performs basic validation on a JSON schema
func ValidateJSONSchema(schemaData json.RawMessage) error {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check for required 'type' field (for basic validation)
	if _, ok := schema["type"]; !ok {
		return fmt.Errorf("schema must have a 'type' field")
	}

	return nil
}
