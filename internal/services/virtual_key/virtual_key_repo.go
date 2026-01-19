package virtual_key

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/curaious/uno/pkg/llm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// VirtualKeyRepo handles database operations for virtual keys
type VirtualKeyRepo struct {
	db *sqlx.DB
}

// NewVirtualKeyRepo creates a new virtual key repository
func NewVirtualKeyRepo(db *sqlx.DB) *VirtualKeyRepo {
	return &VirtualKeyRepo{db: db}
}

// generateSecretKey generates a new secret key with prefix "sk-uno-"
func generateSecretKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64 URL-safe format (without padding)
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)

	// Return with prefix
	return "sk-uno-" + encoded, nil
}

// Create creates a new virtual key with its providers and models
func (r *VirtualKeyRepo) Create(ctx context.Context, req *CreateVirtualKeyRequest) (*VirtualKey, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate secret key
	secretKey, err := generateSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}

	// Create the virtual key
	rateLimits := RateLimits{}
	if req.RateLimits != nil {
		rateLimits = *req.RateLimits
	}

	query := `
		INSERT INTO virtual_keys (name, secret_key, rate_limits)
		VALUES ($1, $2, $3)
		RETURNING id, name, secret_key, rate_limits, created_at, updated_at
	`

	var vk VirtualKey
	err = tx.GetContext(ctx, &vk, query, req.Name, secretKey, rateLimits)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual key: %w", err)
	}

	// Add providers
	if len(req.Providers) > 0 {
		for _, providerType := range req.Providers {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO virtual_key_providers (virtual_key_id, provider_type)
				VALUES ($1, $2)
				ON CONFLICT (virtual_key_id, provider_type) DO NOTHING
			`, vk.ID, providerType)
			if err != nil {
				return nil, fmt.Errorf("failed to add provider: %w", err)
			}
		}
	}

	// Add models
	if len(req.ModelIDs) > 0 {
		for _, modelName := range req.ModelIDs {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO virtual_key_models (virtual_key_id, model_name)
				VALUES ($1, $2)
				ON CONFLICT (virtual_key_id, model_name) DO NOTHING
			`, vk.ID, modelName)
			if err != nil {
				return nil, fmt.Errorf("failed to add model: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load the full virtual key with relationships
	return r.GetByID(ctx, vk.ID)
}

// GetByID retrieves a virtual key by ID with its providers and models
func (r *VirtualKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*VirtualKey, error) {
	// Get the virtual key
	query := `
		SELECT id, name, secret_key, rate_limits, created_at, updated_at
		FROM virtual_keys
		WHERE id = $1
	`

	var vk VirtualKey
	err := r.db.GetContext(ctx, &vk, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual key not found")
		}
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}

	// Get providers
	providerRows, err := r.db.QueryContext(ctx, `
		SELECT provider_type
		FROM virtual_key_providers
		WHERE virtual_key_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}
	defer providerRows.Close()

	var providers []llm.ProviderName
	for providerRows.Next() {
		var pt string
		if err := providerRows.Scan(&pt); err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, llm.ProviderName(pt))
	}
	vk.Providers = providers

	// Get models
	modelRows, err := r.db.QueryContext(ctx, `
		SELECT model_name
		FROM virtual_key_models
		WHERE virtual_key_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer modelRows.Close()

	var modelNames []string
	for modelRows.Next() {
		var modelName string
		if err := modelRows.Scan(&modelName); err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		modelNames = append(modelNames, modelName)
	}
	vk.ModelNames = modelNames

	return &vk, nil
}

// GetByName retrieves a virtual key by name
func (r *VirtualKeyRepo) GetByName(ctx context.Context, name string) (*VirtualKey, error) {
	query := `
		SELECT id, name, secret_key, rate_limits, created_at, updated_at
		FROM virtual_keys
		WHERE name = $1
	`

	var vk VirtualKey
	err := r.db.GetContext(ctx, &vk, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual key not found")
		}
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}

	// Load relationships
	return r.GetByID(ctx, vk.ID)
}

// GetBySecretKey retrieves a virtual key by its secret key
func (r *VirtualKeyRepo) GetBySecretKey(ctx context.Context, secretKey string) (*VirtualKey, error) {
	query := `
		SELECT id, name, secret_key, rate_limits, created_at, updated_at
		FROM virtual_keys
		WHERE secret_key = $1
	`

	var vk VirtualKey
	err := r.db.GetContext(ctx, &vk, query, secretKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("virtual key not found")
		}
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}

	// Load relationships
	return r.GetByID(ctx, vk.ID)
}

// List retrieves all virtual keys with their providers and models
func (r *VirtualKeyRepo) List(ctx context.Context) ([]*VirtualKey, error) {
	query := `
		SELECT id, name, secret_key, rate_limits, created_at, updated_at
		FROM virtual_keys
		ORDER BY created_at DESC
	`

	var virtualKeys []*VirtualKey
	err := r.db.SelectContext(ctx, &virtualKeys, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual keys: %w", err)
	}

	// Load relationships for each virtual key
	for i := range virtualKeys {
		vk, err := r.GetByID(ctx, virtualKeys[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load relationships for virtual key %s: %w", virtualKeys[i].ID, err)
		}
		virtualKeys[i] = vk
	}

	return virtualKeys, nil
}

// Update updates a virtual key and its relationships
func (r *VirtualKeyRepo) Update(ctx context.Context, id uuid.UUID, req *UpdateVirtualKeyRequest) (*VirtualKey, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update name if provided
	if req.Name != nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE virtual_keys
			SET name = $1, updated_at = NOW()
			WHERE id = $2
		`, *req.Name, id)
		if err != nil {
			return nil, fmt.Errorf("failed to update virtual key: %w", err)
		}
	}

	// Update rate limits if provided
	if req.RateLimits != nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE virtual_keys
			SET rate_limits = $1, updated_at = NOW()
			WHERE id = $2
		`, *req.RateLimits, id)
		if err != nil {
			return nil, fmt.Errorf("failed to update rate limits: %w", err)
		}
	}

	// Update providers if provided
	if req.Providers != nil {
		// Delete existing providers
		_, err = tx.ExecContext(ctx, `
			DELETE FROM virtual_key_providers
			WHERE virtual_key_id = $1
		`, id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete providers: %w", err)
		}

		// Add new providers
		for _, providerType := range *req.Providers {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO virtual_key_providers (virtual_key_id, provider_type)
				VALUES ($1, $2)
			`, id, providerType)
			if err != nil {
				return nil, fmt.Errorf("failed to add provider: %w", err)
			}
		}
	}

	// Update models if provided
	if req.ModelIDs != nil {
		// Delete existing models
		_, err = tx.ExecContext(ctx, `
			DELETE FROM virtual_key_models
			WHERE virtual_key_id = $1
		`, id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete models: %w", err)
		}

		// Add new models
		for _, modelName := range *req.ModelIDs {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO virtual_key_models (virtual_key_id, model_name)
				VALUES ($1, $2)
			`, id, modelName)
			if err != nil {
				return nil, fmt.Errorf("failed to add model: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated virtual key
	return r.GetByID(ctx, id)
}

// Delete deletes a virtual key (cascade will delete relationships)
func (r *VirtualKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM virtual_keys
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete virtual key: %w", err)
	}

	return nil
}
