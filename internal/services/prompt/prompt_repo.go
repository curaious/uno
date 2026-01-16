package prompt

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PromptRepo handles database operations for prompts and prompt versions
type PromptRepo struct {
	db *sqlx.DB
}

// NewPromptRepo creates a new prompt repository
func NewPromptRepo(db *sqlx.DB) *PromptRepo {
	return &PromptRepo{db: db}
}

// CreatePrompt creates a new prompt
func (r *PromptRepo) CreatePrompt(ctx context.Context, projectID uuid.UUID, name string) (*Prompt, error) {
	query := `
		INSERT INTO prompts (project_id, name)
		VALUES ($1, $2)
		RETURNING id, project_id, name, created_at, updated_at
	`

	var prompt Prompt
	err := r.db.GetContext(ctx, &prompt, query, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	return &prompt, nil
}

// CreatePromptVersion creates a new prompt version with auto-incremented version number
func (r *PromptRepo) CreatePromptVersion(ctx context.Context, projectID uuid.UUID, promptID uuid.UUID, template, commitMessage string, label *string) (*PromptVersion, error) {
	// Get the next version number
	var nextVersion int
	versionQuery := `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM prompt_versions pv
		JOIN prompts p ON p.id = pv.prompt_id and p.project_id = $1
		WHERE prompt_id = $2
	`
	err := r.db.GetContext(ctx, &nextVersion, versionQuery, projectID, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version number: %w", err)
	}

	// If setting a label, remove it from other versions of the same prompt to avoid unique constraint violation
	if label != nil && *label != "" {
		removeLabelQuery := `
			UPDATE prompt_versions 
			SET label = NULL, updated_at = NOW()
			WHERE prompt_id = $1 AND label = $2
		`
		_, err = r.db.ExecContext(ctx, removeLabelQuery, promptID, *label)
		if err != nil {
			return nil, fmt.Errorf("failed to remove label from other versions: %w", err)
		}
	}

	query := `
		INSERT INTO prompt_versions (prompt_id, version, template, commit_message, label)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, prompt_id, version, template, commit_message, label, created_at, updated_at
	`

	var version PromptVersion
	// Convert empty string label to nil for database
	var labelValue interface{}
	if label != nil && *label != "" {
		labelValue = *label
	} else {
		labelValue = nil
	}

	err = r.db.GetContext(ctx, &version, query, promptID, nextVersion, template, commitMessage, labelValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt version: %w", err)
	}

	return &version, nil
}

// GetPromptByID retrieves a prompt by ID
func (r *PromptRepo) GetPromptByID(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (*Prompt, error) {
	query := `
		SELECT id, project_id, name, created_at, updated_at
		FROM prompts
		WHERE id = $1 AND project_id = $2
	`

	var prompt Prompt
	err := r.db.GetContext(ctx, &prompt, query, id, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt not found")
		}
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	return &prompt, nil
}

// GetPromptByName retrieves a prompt by name
func (r *PromptRepo) GetPromptByName(ctx context.Context, projectID uuid.UUID, name string) (*Prompt, error) {
	query := `
		SELECT id, project_id, name, created_at, updated_at
		FROM prompts
		WHERE name = $1 AND project_id = $2
	`

	var prompt Prompt
	err := r.db.GetContext(ctx, &prompt, query, name, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt not found")
		}
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	return &prompt, nil
}

// GetPromptVersion retrieves a specific prompt version by prompt name and version number
func (r *PromptRepo) GetPromptVersion(ctx context.Context, projectID uuid.UUID, promptName string, version int) (*PromptVersionWithPrompt, error) {
	query := `
		SELECT pv.id, pv.prompt_id, pv.version, pv.template, pv.commit_message, pv.label, pv.created_at, pv.updated_at, p.name as prompt_name
		FROM prompt_versions pv
		JOIN prompts p ON pv.prompt_id = p.id
		WHERE p.name = $1 AND pv.version = $2 AND p.project_id = $3
	`

	var versionWithPrompt PromptVersionWithPrompt
	err := r.db.GetContext(ctx, &versionWithPrompt, query, promptName, version, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version not found")
		}
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	return &versionWithPrompt, nil
}

// GetPromptVersionByID retrives a prompt version by prompt version id
func (r *PromptRepo) GetPromptVersionByID(ctx context.Context, projectID uuid.UUID, promptVersionID uuid.UUID) (*PromptVersionWithPrompt, error) {
	query := `
		SELECT pv.id, pv.prompt_id, pv.version, pv.template, pv.commit_message, pv.label, pv.created_at, pv.updated_at, p.name as prompt_name
		FROM prompt_versions pv
		JOIN prompts p ON pv.prompt_id = p.id
		WHERE pv.id = $1 AND p.project_id = $3
	`

	var versionWithPrompt PromptVersionWithPrompt
	err := r.db.GetContext(ctx, &versionWithPrompt, query, promptVersionID, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version with label not found")
		}
		return nil, fmt.Errorf("failed to get prompt version by label: %w", err)
	}

	return &versionWithPrompt, nil
}

// GetPromptVersionByVersion retrieves a prompt version by prompt id and version
func (r *PromptRepo) GetPromptVersionByVersion(ctx context.Context, projectID uuid.UUID, promptID uuid.UUID, version int) (*PromptVersionWithPrompt, error) {
	query := `
		SELECT pv.id, pv.prompt_id, pv.version, pv.template, pv.commit_message, pv.label, pv.created_at, pv.updated_at, p.name as prompt_name
		FROM prompt_versions pv
		JOIN prompts p ON pv.prompt_id = p.id
		WHERE pv.prompt_id = $1 AND pv.version = $2 AND p.project_id = $3
	`

	var versionWithPrompt PromptVersionWithPrompt
	err := r.db.GetContext(ctx, &versionWithPrompt, query, promptID, version, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version with label not found")
		}
		return nil, fmt.Errorf("failed to get prompt version by label: %w", err)
	}

	return &versionWithPrompt, nil
}

// GetPromptVersionByLabel retrieves a prompt version by prompt name and label
func (r *PromptRepo) GetPromptVersionByLabel(ctx context.Context, projectID uuid.UUID, promptName, label string) (*PromptVersionWithPrompt, error) {
	query := `
		SELECT pv.id, pv.prompt_id, pv.version, pv.template, pv.commit_message, pv.label, pv.created_at, pv.updated_at, p.name as prompt_name
		FROM prompt_versions pv
		JOIN prompts p ON pv.prompt_id = p.id
		WHERE p.name = $1 AND pv.label = $2 AND p.project_id = $3
	`

	var versionWithPrompt PromptVersionWithPrompt
	err := r.db.GetContext(ctx, &versionWithPrompt, query, promptName, label, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version with label not found")
		}
		return nil, fmt.Errorf("failed to get prompt version by label: %w", err)
	}

	return &versionWithPrompt, nil
}

// GetLatestPromptVersion retrieves the latest version of a prompt by name
func (r *PromptRepo) GetLatestPromptVersion(ctx context.Context, projectID uuid.UUID, promptName string) (*PromptVersionWithPrompt, error) {
	query := `
		SELECT pv.id, pv.prompt_id, pv.version, pv.template, pv.commit_message, pv.label, pv.created_at, pv.updated_at, p.name as prompt_name
		FROM prompt_versions pv
		JOIN prompts p ON pv.prompt_id = p.id
		WHERE p.name = $1 AND p.project_id = $2
		ORDER BY pv.version DESC
		LIMIT 1
	`

	var versionWithPrompt PromptVersionWithPrompt
	err := r.db.GetContext(ctx, &versionWithPrompt, query, promptName, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no versions found for prompt")
		}
		return nil, fmt.Errorf("failed to get latest prompt version: %w", err)
	}

	return &versionWithPrompt, nil
}

// ListPrompts retrieves all prompts with their latest version information
func (r *PromptRepo) ListPrompts(ctx context.Context, projectID uuid.UUID) ([]*PromptWithLatestVersion, error) {
	query := `
		SELECT p.id, p.project_id, p.name, p.created_at, p.updated_at,
		       lv.version as latest_version,
		       lv.commit_message as latest_commit_message,
		       lv.label as latest_label
		FROM prompts p
		LEFT JOIN LATERAL (
			SELECT version, commit_message, label
			FROM prompt_versions pv
			WHERE pv.prompt_id = p.id
			ORDER BY pv.version DESC
			LIMIT 1
		) lv ON true
		WHERE p.project_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}
	defer rows.Close()

	var prompts []*PromptWithLatestVersion
	for rows.Next() {
		var prompt PromptWithLatestVersion
		var latestVersion sql.NullInt64
		var latestCommitMsg sql.NullString
		var latestLabel sql.NullString

		err := rows.Scan(
			&prompt.ID,
			&prompt.ProjectID,
			&prompt.Name,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
			&latestVersion,
			&latestCommitMsg,
			&latestLabel,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prompt: %w", err)
		}

		if latestVersion.Valid {
			version := int(latestVersion.Int64)
			prompt.LatestVersion = &version
		}
		if latestCommitMsg.Valid {
			prompt.LatestCommitMsg = &latestCommitMsg.String
		}
		if latestLabel.Valid {
			prompt.LatestLabel = &latestLabel.String
		}

		prompts = append(prompts, &prompt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate prompts: %w", err)
	}

	return prompts, nil
}

// ListPromptVersions retrieves all versions for a specific prompt
func (r *PromptRepo) ListPromptVersions(ctx context.Context, projectID uuid.UUID, promptID uuid.UUID) ([]*PromptVersion, error) {
	query := `
		SELECT pv.id, pv.prompt_id, version, template, commit_message, label, pv.created_at, pv.updated_at
		FROM prompt_versions pv
		JOIN prompts p ON pv.prompt_id = p.id
		WHERE pv.prompt_id = $1 AND p.project_id = $2
		ORDER BY version DESC
	`

	var versions []*PromptVersion
	err := r.db.SelectContext(ctx, &versions, query, promptID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompt versions: %w", err)
	}

	return versions, nil
}

// UpdatePromptVersionLabel updates the label of a prompt version
func (r *PromptRepo) UpdatePromptVersionLabel(ctx context.Context, projectID uuid.UUID, versionID uuid.UUID, label *string) (*PromptVersion, error) {
	// First, get the prompt_id for this version
	var promptID uuid.UUID
	getPromptQuery := `SELECT prompt_id FROM prompt_versions WHERE id = $1`
	err := r.db.GetContext(ctx, &promptID, getPromptQuery, versionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version not found")
		}
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	// If setting a label, remove it from other versions of the same prompt
	if label != nil {
		removeLabelQuery := `
			UPDATE prompt_versions 
			SET label = NULL, updated_at = NOW()
			WHERE prompt_id = $1 AND label = $2 AND id != $3
		`
		_, err = r.db.ExecContext(ctx, removeLabelQuery, promptID, *label, versionID)
		if err != nil {
			return nil, fmt.Errorf("failed to remove label from other versions: %w", err)
		}
	}

	// Update the label
	query := `
		UPDATE prompt_versions
		SET label = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, prompt_id, version, template, commit_message, label, created_at, updated_at
	`

	var version PromptVersion
	err = r.db.GetContext(ctx, &version, query, label, versionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version not found")
		}
		return nil, fmt.Errorf("failed to update prompt version label: %w", err)
	}

	return &version, nil
}

// DeletePromptVersion deletes a specific prompt version
func (r *PromptRepo) DeletePromptVersion(ctx context.Context, versionID uuid.UUID) error {
	query := `DELETE FROM prompt_versions pv WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, versionID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("prompt version not found")
	}

	return nil
}

// DeletePrompt deletes a prompt and all its versions (cascade delete)
func (r *PromptRepo) DeletePrompt(ctx context.Context, projectID uuid.UUID, promptID uuid.UUID) error {
	query := `DELETE FROM prompts WHERE id = $1 AND project_id = $2`

	result, err := r.db.ExecContext(ctx, query, promptID, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("prompt not found")
	}

	return nil
}
