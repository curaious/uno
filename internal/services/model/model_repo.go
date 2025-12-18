package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ModelRepo handles database operations for models
type ModelRepo struct {
	db *sqlx.DB
}

// NewModelRepo creates a new model repository
func NewModelRepo(db *sqlx.DB) *ModelRepo {
	return &ModelRepo{db: db}
}

// Create creates a new model
func (r *ModelRepo) Create(ctx context.Context, projectID uuid.UUID, req *CreateModelRequest) (*Model, error) {
	query := `
		INSERT INTO models (project_id, provider_type, name, model_id, parameters)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, provider_type, name, model_id, parameters, created_at, updated_at
	`

	var m Model
	params := req.Parameters
	if params == nil {
		params = make(ModelParameters)
	}
	err := r.db.GetContext(ctx, &m, query,
		projectID,
		req.ProviderType,
		req.Name,
		req.ModelID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	return &m, nil
}

// GetByID retrieves a model by ID
func (r *ModelRepo) GetByID(ctx context.Context, projectID, id uuid.UUID) (*Model, error) {
	query := `
		SELECT id, project_id, provider_type, name, model_id, parameters, created_at, updated_at
		FROM models
		WHERE id = $1 AND project_id = $2
	`

	var m Model
	err := r.db.GetContext(ctx, &m, query, id, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model not found")
		}
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return &m, nil
}

// GetByName retrieves a model by name
func (r *ModelRepo) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*Model, error) {
	query := `
		SELECT id, project_id, provider_type, name, model_id, parameters, created_at, updated_at
		FROM models
		WHERE name = $1 AND project_id = $2
	`

	var m Model
	err := r.db.GetContext(ctx, &m, query, name, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model not found")
		}
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return &m, nil
}

// List retrieves all models with optional filtering by provider_type
func (r *ModelRepo) List(ctx context.Context, projectID uuid.UUID, providerType *string) ([]*ModelWithProvider, error) {
	query := `
		SELECT 
			m.id, m.project_id, m.provider_type, m.name, m.model_id, 
			m.parameters, m.created_at, m.updated_at,
			m.provider_type as provider_type
		FROM models m
		WHERE m.project_id = $1
	`
	args := []interface{}{projectID}

	if providerType != nil {
		query += " AND m.provider_type = $2"
		args = append(args, *providerType)
	}

	query += " ORDER BY m.created_at DESC"

	var models []*ModelWithProvider
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	return models, nil
}

// Update updates a model
func (r *ModelRepo) Update(ctx context.Context, projectID, id uuid.UUID, req *UpdateModelRequest) (*Model, error) {
	// Build dynamic query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.ProviderType != nil {
		setParts = append(setParts, fmt.Sprintf("provider_type = $%d", argIndex))
		args = append(args, *req.ProviderType)
		argIndex++
	}

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.ModelID != nil {
		setParts = append(setParts, fmt.Sprintf("model_id = $%d", argIndex))
		args = append(args, *req.ModelID)
		argIndex++
	}

	if req.Parameters != nil {
		setParts = append(setParts, fmt.Sprintf("parameters = $%d", argIndex))
		args = append(args, *req.Parameters)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, projectID, id)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, projectID)
	args = append(args, id)
	argIndex += 1

	// Join all set parts with commas
	setClause := ""
	for i, part := range setParts {
		if i > 0 {
			setClause += ", "
		}
		setClause += part
	}

	query := fmt.Sprintf(`
		UPDATE models
		SET %s
		WHERE id = $%d AND project_id = $%d
		RETURNING id, project_id, provider_type, name, model_id, parameters, created_at, updated_at
	`, setClause, argIndex, argIndex-1)

	var m Model
	err := r.db.GetContext(ctx, &m, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model not found")
		}
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	return &m, nil
}

// Delete deletes a model
func (r *ModelRepo) Delete(ctx context.Context, projectID, id uuid.UUID) error {
	query := `DELETE FROM models WHERE id = $1 AND project_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("model not found")
	}

	return nil
}
