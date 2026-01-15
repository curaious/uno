package agent_config

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AgentConfigRepo handles database operations for agent configs
type AgentConfigRepo struct {
	db *sqlx.DB
}

// NewAgentConfigRepo creates a new agent config repository
func NewAgentConfigRepo(db *sqlx.DB) *AgentConfigRepo {
	return &AgentConfigRepo{db: db}
}

// Create creates a new agent config with version 1
func (r *AgentConfigRepo) Create(ctx context.Context, projectID uuid.UUID, req *CreateAgentConfigRequest) (*AgentConfig, error) {
	query := `
		INSERT INTO agent_configs (project_id, name, version, config)
		VALUES ($1, $2, 1, $3)
		RETURNING id, project_id, name, version, config, created_at, updated_at
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, projectID, req.Name, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config: %w", err)
	}

	return &config, nil
}

// CreateVersion creates a new version of an existing agent config
func (r *AgentConfigRepo) CreateVersion(ctx context.Context, projectID uuid.UUID, name string, req *UpdateAgentConfigRequest) (*AgentConfig, error) {
	// Get the latest version number
	var latestVersion int
	err := r.db.GetContext(ctx, &latestVersion, `
		SELECT COALESCE(MAX(version), 0) FROM agent_configs WHERE project_id = $1 AND name = $2
	`, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	newVersion := latestVersion + 1

	query := `
		INSERT INTO agent_configs (project_id, name, version, config)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, name, version, config, created_at, updated_at
	`

	var config AgentConfig
	err = r.db.GetContext(ctx, &config, query, projectID, name, newVersion, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config version: %w", err)
	}

	return &config, nil
}

// GetByID retrieves an agent config by ID
func (r *AgentConfigRepo) GetByID(ctx context.Context, projectID, id uuid.UUID) (*AgentConfig, error) {
	query := `
		SELECT id, project_id, name, version, config, created_at, updated_at
		FROM agent_configs
		WHERE id = $1 AND project_id = $2
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, id, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent config not found")
		}
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return &config, nil
}

// GetByNameAndVersion retrieves an agent config by name and version
func (r *AgentConfigRepo) GetByNameAndVersion(ctx context.Context, projectID uuid.UUID, name string, version int) (*AgentConfig, error) {
	query := `
		SELECT id, project_id, name, version, config, created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1 AND name = $2 AND version = $3
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, projectID, name, version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent config not found")
		}
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return &config, nil
}

// GetLatestByName retrieves the latest version of an agent config by name
func (r *AgentConfigRepo) GetLatestByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentConfig, error) {
	query := `
		SELECT id, project_id, name, version, config, created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1 AND name = $2
		ORDER BY version DESC
		LIMIT 1
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, projectID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent config not found")
		}
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return &config, nil
}

// List retrieves all unique agent configs (latest version summary) for a project
func (r *AgentConfigRepo) List(ctx context.Context, projectID uuid.UUID) ([]*AgentConfigSummary, error) {
	query := `
		SELECT DISTINCT ON (name)
			id, project_id, name, version as latest_version, created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1
		ORDER BY name, version DESC
	`

	var configs []*AgentConfigSummary
	err := r.db.SelectContext(ctx, &configs, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent configs: %w", err)
	}

	return configs, nil
}

// ListVersions retrieves all versions of an agent config by name
func (r *AgentConfigRepo) ListVersions(ctx context.Context, projectID uuid.UUID, name string) ([]*AgentConfig, error) {
	query := `
		SELECT id, project_id, name, version, config, created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1 AND name = $2
		ORDER BY version DESC
	`

	var configs []*AgentConfig
	err := r.db.SelectContext(ctx, &configs, query, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent config versions: %w", err)
	}

	return configs, nil
}

// Delete deletes all versions of an agent config by name
func (r *AgentConfigRepo) Delete(ctx context.Context, projectID uuid.UUID, name string) error {
	query := `DELETE FROM agent_configs WHERE project_id = $1 AND name = $2`
	result, err := r.db.ExecContext(ctx, query, projectID, name)
	if err != nil {
		return fmt.Errorf("failed to delete agent config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("agent config not found")
	}

	return nil
}

// DeleteVersion deletes a specific version of an agent config
func (r *AgentConfigRepo) DeleteVersion(ctx context.Context, projectID uuid.UUID, name string, version int) error {
	query := `DELETE FROM agent_configs WHERE project_id = $1 AND name = $2 AND version = $3`
	result, err := r.db.ExecContext(ctx, query, projectID, name, version)
	if err != nil {
		return fmt.Errorf("failed to delete agent config version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("agent config version not found")
	}

	return nil
}

// Exists checks if an agent config with the given name exists
func (r *AgentConfigRepo) Exists(ctx context.Context, projectID uuid.UUID, name string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM agent_configs WHERE project_id = $1 AND name = $2
	`, projectID, name)
	if err != nil {
		return false, fmt.Errorf("failed to check agent config existence: %w", err)
	}
	return count > 0, nil
}
