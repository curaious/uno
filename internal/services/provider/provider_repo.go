package provider

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/praveen001/uno/pkg/llm"
)

// ProviderRepo handles database operations for API keys
type ProviderRepo struct {
	db *sqlx.DB
}

// NewProviderRepo creates a new provider repository
func NewProviderRepo(db *sqlx.DB) *ProviderRepo {
	return &ProviderRepo{db: db}
}

// Create creates a new API key
func (r *ProviderRepo) Create(ctx context.Context, req *CreateAPIKeyRequest) (*APIKey, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// If this is set as default, unset other defaults for this provider type
	if req.IsDefault {
		_, err = tx.ExecContext(ctx, `
			UPDATE api_keys
			SET is_default = false, updated_at = NOW()
			WHERE provider_type = $1 AND is_default = true
		`, req.ProviderType)
		if err != nil {
			return nil, fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	enabled := true
	if req.Enabled != enabled {
		enabled = req.Enabled
	}

	query := `
		INSERT INTO api_keys (provider_type, name, api_key, enabled, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
	`

	var apiKey APIKey
	err = tx.GetContext(ctx, &apiKey, query,
		req.ProviderType, req.Name, req.APIKey, enabled, req.IsDefault)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &apiKey, nil
}

// GetByID retrieves an API key by ID
func (r *ProviderRepo) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	query := `
		SELECT id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
		FROM api_keys
		WHERE id = $1
	`

	var apiKey APIKey
	err := r.db.GetContext(ctx, &apiKey, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// GetByName retrieves an API key by provider type and name
func (r *ProviderRepo) GetByName(ctx context.Context, providerType llm.ProviderName, name string) (*APIKey, error) {
	query := `
		SELECT id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
		FROM api_keys
		WHERE provider_type = $1 AND name = $2
	`

	var apiKey APIKey
	err := r.db.GetContext(ctx, &apiKey, query, providerType, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// List retrieves all API keys with optional filtering
func (r *ProviderRepo) List(ctx context.Context, providerType *llm.ProviderName, enabledOnly bool) ([]*APIKey, error) {
	query := `
		SELECT id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
		FROM api_keys
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if providerType != nil {
		query += fmt.Sprintf(" AND provider_type = $%d", argIndex)
		args = append(args, *providerType)
		argIndex++
	}

	if enabledOnly {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, true)
		argIndex++
	}

	query += " ORDER BY provider_type, is_default DESC, created_at ASC"

	var apiKeys []*APIKey
	err := r.db.SelectContext(ctx, &apiKeys, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return apiKeys, nil
}

// GetDefaultAPIKey retrieves the default API key for a provider type
func (r *ProviderRepo) GetDefaultAPIKey(ctx context.Context, providerType llm.ProviderName) (*APIKey, error) {
	query := `
		SELECT id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
		FROM api_keys
		WHERE provider_type = $1 AND enabled = true AND is_default = true
		LIMIT 1
	`

	var apiKey APIKey
	err := r.db.GetContext(ctx, &apiKey, query, providerType)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no default, get the first enabled one
			query = `
				SELECT id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
				FROM api_keys
				WHERE provider_type = $1 AND enabled = true
				ORDER BY created_at ASC
				LIMIT 1
			`
			err = r.db.GetContext(ctx, &apiKey, query, providerType)
			if err != nil {
				return nil, fmt.Errorf("no enabled API key found for provider type: %s", providerType)
			}
		} else {
			return nil, fmt.Errorf("failed to get default API key: %w", err)
		}
	}

	return &apiKey, nil
}

// Update updates an API key
func (r *ProviderRepo) Update(ctx context.Context, id uuid.UUID, req *UpdateAPIKeyRequest) (*APIKey, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.APIKey != nil {
		setParts = append(setParts, fmt.Sprintf("api_key = $%d", argIndex))
		args = append(args, *req.APIKey)
		argIndex++
	}

	if req.Enabled != nil {
		setParts = append(setParts, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, *req.Enabled)
		argIndex++
	}

	if req.IsDefault != nil {
		if *req.IsDefault {
			// Get the provider type first
			var providerType llm.ProviderName
			err = tx.GetContext(ctx, &providerType, "SELECT provider_type FROM api_keys WHERE id = $1", id)
			if err != nil {
				return nil, fmt.Errorf("failed to get provider type: %w", err)
			}

			// Unset other defaults for this provider type
			_, err = tx.ExecContext(ctx, `
				UPDATE api_keys
				SET is_default = false, updated_at = NOW()
				WHERE provider_type = $1 AND is_default = true AND id != $2
			`, providerType, id)
			if err != nil {
				return nil, fmt.Errorf("failed to unset other defaults: %w", err)
			}
		}
		setParts = append(setParts, fmt.Sprintf("is_default = $%d", argIndex))
		args = append(args, *req.IsDefault)
		argIndex++
	}

	if len(setParts) == 0 {
		tx.Commit()
		return r.GetByID(ctx, id)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	setClause := ""
	for i, part := range setParts {
		if i > 0 {
			setClause += ", "
		}
		setClause += part
	}

	query := fmt.Sprintf(`
		UPDATE api_keys
		SET %s
		WHERE id = $%d
		RETURNING id, provider_type, name, api_key, enabled, is_default, created_at, updated_at
	`, setClause, argIndex)

	var apiKey APIKey
	err = tx.GetContext(ctx, &apiKey, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &apiKey, nil
}

// Delete deletes an API key
func (r *ProviderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// GetProviderConfig retrieves provider config by provider type
func (r *ProviderRepo) GetProviderConfig(ctx context.Context, providerType llm.ProviderName) (*ProviderConfig, error) {
	query := `
		SELECT provider_type, base_url, custom_headers, created_at, updated_at
		FROM provider_configs
		WHERE provider_type = $1
	`

	var config ProviderConfig
	err := r.db.GetContext(ctx, &config, query, providerType)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty config if not found
			return &ProviderConfig{
				ProviderType:  providerType,
				BaseURL:       nil,
				CustomHeaders: make(CustomHeadersMap),
			}, nil
		}
		return nil, fmt.Errorf("failed to get provider config: %w", err)
	}

	return &config, nil
}

// CreateOrUpdateProviderConfig creates or updates provider config
func (r *ProviderRepo) CreateOrUpdateProviderConfig(ctx context.Context, req *CreateProviderConfigRequest) (*ProviderConfig, error) {
	var customHeaders CustomHeadersMap
	if req.CustomHeaders != nil {
		customHeaders = req.CustomHeaders
	} else {
		customHeaders = make(CustomHeadersMap)
	}

	query := `
		INSERT INTO provider_configs (provider_type, base_url, custom_headers)
		VALUES ($1, $2, $3)
		ON CONFLICT (provider_type) 
		DO UPDATE SET 
			base_url = EXCLUDED.base_url,
			custom_headers = EXCLUDED.custom_headers,
			updated_at = NOW()
		RETURNING provider_type, base_url, custom_headers, created_at, updated_at
	`

	var config ProviderConfig
	err := r.db.GetContext(ctx, &config, query, req.ProviderType, req.BaseURL, customHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update provider config: %w", err)
	}

	return &config, nil
}

// UpdateProviderConfig updates provider config
func (r *ProviderRepo) UpdateProviderConfig(ctx context.Context, providerType llm.ProviderName, req *UpdateProviderConfigRequest) (*ProviderConfig, error) {
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.BaseURL != nil {
		var baseURLValue interface{}
		if *req.BaseURL == "" {
			baseURLValue = nil
		} else {
			baseURLValue = *req.BaseURL
		}
		setParts = append(setParts, fmt.Sprintf("base_url = $%d", argIndex))
		args = append(args, baseURLValue)
		argIndex++
	}

	if req.CustomHeaders != nil {
		var headersValue interface{}
		if len(*req.CustomHeaders) == 0 {
			headersValue = make(CustomHeadersMap)
		} else {
			headersValue = *req.CustomHeaders
		}
		setParts = append(setParts, fmt.Sprintf("custom_headers = $%d", argIndex))
		args = append(args, headersValue)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetProviderConfig(ctx, providerType)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, providerType)

	setClause := ""
	for i, part := range setParts {
		if i > 0 {
			setClause += ", "
		}
		setClause += part
	}

	query := fmt.Sprintf(`
		UPDATE provider_configs
		SET %s
		WHERE provider_type = $%d
		RETURNING provider_type, base_url, custom_headers, created_at, updated_at
	`, setClause, argIndex)

	var config ProviderConfig
	err := r.db.GetContext(ctx, &config, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider config not found")
		}
		return nil, fmt.Errorf("failed to update provider config: %w", err)
	}

	return &config, nil
}

// ListProviderConfigs retrieves all provider configs
func (r *ProviderRepo) ListProviderConfigs(ctx context.Context) ([]*ProviderConfig, error) {
	query := `
		SELECT provider_type, base_url, custom_headers, created_at, updated_at
		FROM provider_configs
		ORDER BY provider_type
	`

	var configs []*ProviderConfig
	err := r.db.SelectContext(ctx, &configs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider configs: %w", err)
	}

	return configs, nil
}
