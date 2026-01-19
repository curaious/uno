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

// Create creates a new agent config with agent_id and version 0
func (r *AgentConfigRepo) Create(ctx context.Context, projectID uuid.UUID, req *CreateAgentConfigRequest) (*AgentConfig, error) {
	agentID := uuid.New() // Generate stable UUID for this agent

	query := `
		INSERT INTO agent_configs (agent_id, project_id, name, version, immutable, config)
		VALUES ($1, $2, $3, 0, false, $4)
		RETURNING id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, agentID, projectID, req.Name, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config: %w", err)
	}

	return &config, nil
}

// UpdateVersion0 updates version 0 in place (mutable)
func (r *AgentConfigRepo) UpdateVersion0(ctx context.Context, agentID uuid.UUID, req *UpdateAgentConfigRequest) (*AgentConfig, error) {
	query := `
		UPDATE agent_configs 
		SET config = $1, updated_at = NOW()
		WHERE agent_id = $2 AND version = 0
		RETURNING id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, req.Config, agentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent config version 0 not found")
		}
		return nil, fmt.Errorf("failed to update agent config: %w", err)
	}

	return &config, nil
}

// CreateVersion creates a new immutable version from version 0
func (r *AgentConfigRepo) CreateVersion(ctx context.Context, agentID uuid.UUID) (*AgentConfig, error) {
	// Get version 0 config
	var version0 AgentConfig
	err := r.db.GetContext(ctx, &version0, `
		SELECT id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
		FROM agent_configs
		WHERE agent_id = $1 AND version = 0
	`, agentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent config version 0 not found")
		}
		return nil, fmt.Errorf("failed to get version 0: %w", err)
	}

	// Get the latest version number
	var latestVersion int
	err = r.db.GetContext(ctx, &latestVersion, `
		SELECT COALESCE(MAX(version), 0) FROM agent_configs WHERE agent_id = $1
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	newVersion := latestVersion + 1

	// Create new immutable version
	query := `
		INSERT INTO agent_configs (agent_id, project_id, name, version, immutable, config)
		VALUES ($1, $2, $3, $4, true, $5)
		RETURNING id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
	`

	var config AgentConfig
	err = r.db.GetContext(ctx, &config, query, agentID, version0.ProjectID, version0.Name, newVersion, version0.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config version: %w", err)
	}

	return &config, nil
}

// GetByID retrieves an agent config by row ID
func (r *AgentConfigRepo) GetByID(ctx context.Context, projectID, id uuid.UUID) (*AgentConfig, error) {
	query := `
		SELECT id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
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

// GetByAgentIDAndVersion retrieves an agent config by agent_id and version
func (r *AgentConfigRepo) GetByAgentIDAndVersion(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID, version int) (*AgentConfig, error) {
	query := `
		SELECT id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1 AND agent_id = $2 AND version = $3
	`

	var config AgentConfig
	err := r.db.GetContext(ctx, &config, query, projectID, agentID, version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent config not found")
		}
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return &config, nil
}

// GetByNameAndVersion retrieves an agent config by name and version (for backward compatibility)
func (r *AgentConfigRepo) GetByNameAndVersion(ctx context.Context, projectID uuid.UUID, name string, version int) (*AgentConfig, error) {
	query := `
		SELECT id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
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

// GetLatestByName retrieves version 0 (the mutable version) of an agent config by name
func (r *AgentConfigRepo) GetLatestByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentConfig, error) {
	query := `
		SELECT id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1 AND name = $2 AND version = 0
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

// GetAgentIDByName retrieves the agent_id for a given name
func (r *AgentConfigRepo) GetAgentIDByName(ctx context.Context, projectID uuid.UUID, name string) (uuid.UUID, error) {
	var agentID uuid.UUID
	err := r.db.GetContext(ctx, &agentID, `
		SELECT agent_id FROM agent_configs 
		WHERE project_id = $1 AND name = $2 AND version = 0
		LIMIT 1
	`, projectID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, fmt.Errorf("agent config not found")
		}
		return uuid.Nil, fmt.Errorf("failed to get agent_id: %w", err)
	}
	return agentID, nil
}

// List retrieves all unique agent configs (version 0 summary) for a project
func (r *AgentConfigRepo) List(ctx context.Context, projectID uuid.UUID) ([]*AgentConfigSummary, error) {
	query := `
		SELECT id, agent_id, project_id, name, 
		       COALESCE(MAX(version) FILTER (WHERE version > 0), 0) as latest_version,
		       created_at, updated_at
		FROM agent_configs
		WHERE project_id = $1 AND version = 0
		GROUP BY id, agent_id, project_id, name, created_at, updated_at
		ORDER BY name
	`

	var configs []*AgentConfigSummary
	err := r.db.SelectContext(ctx, &configs, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent configs: %w", err)
	}

	return configs, nil
}

// ListVersions retrieves all versions of an agent config by agent_id
func (r *AgentConfigRepo) ListVersions(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) ([]*AgentConfig, error) {
	query := `
		SELECT id, agent_id, project_id, name, version, immutable, config, created_at, updated_at
		FROM agent_configs
		WHERE agent_id = $1 and project_id = $2
		ORDER BY version DESC
	`

	var configs []*AgentConfig
	err := r.db.SelectContext(ctx, &configs, query, agentID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent config versions: %w", err)
	}

	return configs, nil
}

// ListVersionsByName retrieves all versions of an agent config by name (for backward compatibility)
func (r *AgentConfigRepo) ListVersionsByName(ctx context.Context, projectID uuid.UUID, name string) ([]*AgentConfig, error) {
	// First get agent_id
	agentID, err := r.GetAgentIDByName(ctx, projectID, name)
	if err != nil {
		return nil, err
	}
	return r.ListVersions(ctx, projectID, agentID)
}

// Delete deletes all versions of an agent config by agent_id
func (r *AgentConfigRepo) Delete(ctx context.Context, agentID uuid.UUID) error {
	query := `DELETE FROM agent_configs WHERE agent_id = $1`
	result, err := r.db.ExecContext(ctx, query, agentID)
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

// DeleteByName deletes all versions of an agent config by name
func (r *AgentConfigRepo) DeleteByName(ctx context.Context, projectID uuid.UUID, name string) error {
	agentID, err := r.GetAgentIDByName(ctx, projectID, name)
	if err != nil {
		return err
	}
	return r.Delete(ctx, agentID)
}

// DeleteVersion deletes a specific version of an agent config
func (r *AgentConfigRepo) DeleteVersion(ctx context.Context, agentID uuid.UUID, version int) error {
	// Prevent deletion of version 0
	if version == 0 {
		return fmt.Errorf("cannot delete version 0")
	}

	query := `DELETE FROM agent_configs WHERE agent_id = $1 AND version = $2`
	result, err := r.db.ExecContext(ctx, query, agentID, version)
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
		SELECT COUNT(*) FROM agent_configs WHERE project_id = $1 AND name = $2 AND version = 0
	`, projectID, name)
	if err != nil {
		return false, fmt.Errorf("failed to check agent config existence: %w", err)
	}
	return count > 0, nil
}

// CreateAlias creates a new alias for an agent config
func (r *AgentConfigRepo) CreateAlias(ctx context.Context, projectID, agentID uuid.UUID, req *CreateAliasRequest) (*AgentConfigAlias, error) {
	// Validate that version1 exists
	_, err := r.GetByAgentIDAndVersion(ctx, projectID, agentID, req.Version1)
	if err != nil {
		return nil, fmt.Errorf("version1 %d does not exist: %w", req.Version1, err)
	}

	// Validate that version2 exists if provided
	if req.Version2 != nil {
		_, err := r.GetByAgentIDAndVersion(ctx, projectID, agentID, *req.Version2)
		if err != nil {
			return nil, fmt.Errorf("version2 %d does not exist: %w", *req.Version2, err)
		}
		// Validate weight is provided if version2 is set
		if req.Weight == nil {
			return nil, fmt.Errorf("weight is required when version2 is set")
		}
	}

	query := `
		INSERT INTO agent_config_aliases (project_id, agent_id, name, version1, version2, weight)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, project_id, agent_id, name, version1, version2, weight, created_at, updated_at
	`

	var alias AgentConfigAlias
	err = r.db.GetContext(ctx, &alias, query, projectID, agentID, req.Name, req.Version1, req.Version2, req.Weight)
	if err != nil {
		return nil, fmt.Errorf("failed to create alias: %w", err)
	}

	return &alias, nil
}

// GetAlias retrieves an alias by ID
func (r *AgentConfigRepo) GetAlias(ctx context.Context, projectID, id uuid.UUID) (*AgentConfigAlias, error) {
	query := `
		SELECT id, project_id, agent_id, name, version1, version2, weight, created_at, updated_at
		FROM agent_config_aliases
		WHERE id = $1 AND project_id = $2
	`

	var alias AgentConfigAlias
	err := r.db.GetContext(ctx, &alias, query, id, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alias not found")
		}
		return nil, fmt.Errorf("failed to get alias: %w", err)
	}

	return &alias, nil
}

// GetAliasByName retrieves an alias by agent_id and name
func (r *AgentConfigRepo) GetAliasByName(ctx context.Context, projectID, agentID uuid.UUID, name string) (*AgentConfigAlias, error) {
	query := `
		SELECT id, project_id, agent_id, name, version1, version2, weight, created_at, updated_at
		FROM agent_config_aliases
		WHERE project_id = $1 AND agent_id = $2 AND name = $3
	`

	var alias AgentConfigAlias
	err := r.db.GetContext(ctx, &alias, query, projectID, agentID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alias not found")
		}
		return nil, fmt.Errorf("failed to get alias: %w", err)
	}

	return &alias, nil
}

// ListAliases retrieves all aliases for an agent
func (r *AgentConfigRepo) ListAliases(ctx context.Context, projectID, agentID uuid.UUID) ([]*AgentConfigAlias, error) {
	query := `
		SELECT id, project_id, agent_id, name, version1, version2, weight, created_at, updated_at
		FROM agent_config_aliases
		WHERE project_id = $1 AND agent_id = $2
		ORDER BY name
	`

	var aliases []*AgentConfigAlias
	err := r.db.SelectContext(ctx, &aliases, query, projectID, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list aliases: %w", err)
	}

	return aliases, nil
}

// UpdateAlias updates an existing alias
func (r *AgentConfigRepo) UpdateAlias(ctx context.Context, projectID, id uuid.UUID, req *UpdateAliasRequest) (*AgentConfigAlias, error) {
	// Get existing alias to get agent_id
	existing, err := r.GetAlias(ctx, projectID, id)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, req.Name)
		argIndex++
	}

	if req.Version1 != nil {
		// Validate that version1 exists
		_, err := r.GetByAgentIDAndVersion(ctx, projectID, existing.AgentID, *req.Version1)
		if err != nil {
			return nil, fmt.Errorf("version1 %d does not exist: %w", *req.Version1, err)
		}
		updates = append(updates, fmt.Sprintf("version1 = $%d", argIndex))
		args = append(args, *req.Version1)
		argIndex++
	}

	if req.Version2 != nil {
		// Validate that version2 exists
		_, err := r.GetByAgentIDAndVersion(ctx, projectID, existing.AgentID, *req.Version2)
		if err != nil {
			return nil, fmt.Errorf("version2 %d does not exist: %w", *req.Version2, err)
		}
		updates = append(updates, fmt.Sprintf("version2 = $%d", argIndex))
		args = append(args, *req.Version2)
		argIndex++
		// If version2 is set, weight must be provided
		if req.Weight == nil {
			return nil, fmt.Errorf("weight is required when version2 is set")
		}
		updates = append(updates, fmt.Sprintf("weight = $%d", argIndex))
		args = append(args, *req.Weight)
		argIndex++
	} else if req.Version2 == nil && req.Weight != nil {
		// If version2 is being cleared, weight should also be cleared
		updates = append(updates, fmt.Sprintf("version2 = NULL"))
		updates = append(updates, fmt.Sprintf("weight = NULL"))
	}

	if len(updates) == 0 {
		return existing, nil // No changes
	}

	updates = append(updates, "updated_at = NOW()")
	setClause := ""
	for i, update := range updates {
		if i > 0 {
			setClause += ", "
		}
		setClause += update
	}

	query := fmt.Sprintf(`
		UPDATE agent_config_aliases
		SET %s
		WHERE id = $%d AND project_id = $%d
		RETURNING id, project_id, agent_id, name, version1, version2, weight, created_at, updated_at
	`, setClause, argIndex, argIndex+1)

	args = append(args, id, projectID)

	var alias AgentConfigAlias
	err = r.db.GetContext(ctx, &alias, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alias not found")
		}
		return nil, fmt.Errorf("failed to update alias: %w", err)
	}

	return &alias, nil
}

// DeleteAlias deletes an alias by ID
func (r *AgentConfigRepo) DeleteAlias(ctx context.Context, projectID, id uuid.UUID) error {
	query := `DELETE FROM agent_config_aliases WHERE id = $1 AND project_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete alias: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alias not found")
	}

	return nil
}
