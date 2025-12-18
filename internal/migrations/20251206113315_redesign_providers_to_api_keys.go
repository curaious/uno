package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251206113315",
		up:      mig_20251206113315_redesign_providers_to_api_keys_up,
		down:    mig_20251206113315_redesign_providers_to_api_keys_down,
	})
}

func mig_20251206113315_redesign_providers_to_api_keys_up(tx *sqlx.Tx) error {
	// Create new api_keys table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			provider_type VARCHAR(50) NOT NULL,
			name VARCHAR(255) NOT NULL,
			api_key TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true,
			is_default BOOLEAN DEFAULT false,
			base_url VARCHAR(500),
			custom_headers JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(provider_type, name)
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_api_keys_provider_type ON api_keys(provider_type);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);
	`)
	if err != nil {
		return err
	}

	// Create unique index to ensure only one default key per provider type
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_provider_default 
		ON api_keys(provider_type, is_default) 
		WHERE is_default = true;
	`)
	if err != nil {
		return err
	}

	// Migrate data from providers and provider_api_keys to api_keys
	// Map old provider types to new ones
	// First, migrate providers with their default API keys
	_, err = tx.Exec(`
		INSERT INTO api_keys (provider_type, name, api_key, enabled, is_default, base_url, custom_headers)
		SELECT 
			CASE 
				WHEN p.provider = 'openai' THEN 'OpenAI'
				WHEN p.provider = 'anthropic' THEN 'Anthropic'
				WHEN p.provider = 'google' THEN 'Gemini'
				WHEN p.provider = 'xai' THEN 'xAI'
				ELSE p.provider::text
			END as provider_type,
			COALESCE(pak.api_key, '') as name,
			COALESCE(pak.api_key, '') as api_key,
			true as enabled,
			COALESCE(pak.is_default, true) as is_default,
			p.base_url,
			p.custom_headers
		FROM providers p
		LEFT JOIN provider_api_keys pak ON p.id = pak.provider_id AND pak.is_default = true
		WHERE EXISTS (SELECT 1 FROM provider_api_keys WHERE provider_id = p.id)
		OR pak.api_key IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Migrate non-default API keys
	_, err = tx.Exec(`
		INSERT INTO api_keys (provider_type, name, api_key, enabled, is_default, base_url, custom_headers)
		SELECT 
			CASE 
				WHEN p.provider = 'openai' THEN 'OpenAI'
				WHEN p.provider = 'anthropic' THEN 'Anthropic'
				WHEN p.provider = 'google' THEN 'Gemini'
				WHEN p.provider = 'xai' THEN 'xAI'
				ELSE p.provider::text
			END as provider_type,
			'Key ' || ROW_NUMBER() OVER (PARTITION BY p.id ORDER BY pak.created_at) as name,
			pak.api_key,
			true as enabled,
			pak.is_default,
			p.base_url,
			p.custom_headers
		FROM providers p
		JOIN provider_api_keys pak ON p.id = pak.provider_id
		WHERE pak.is_default = false
		ON CONFLICT (provider_type, name) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// Update models table to use provider_type instead of provider_id
	// First, add provider_type column
	_, err = tx.Exec(`
		ALTER TABLE models
		ADD COLUMN IF NOT EXISTS provider_type VARCHAR(50);
	`)
	if err != nil {
		return err
	}

	// Migrate provider_id to provider_type with proper mapping
	_, err = tx.Exec(`
		UPDATE models m
		SET provider_type = CASE 
			WHEN p.provider = 'openai' THEN 'OpenAI'
			WHEN p.provider = 'anthropic' THEN 'Anthropic'
			WHEN p.provider = 'google' THEN 'Gemini'
			WHEN p.provider = 'xai' THEN 'xAI'
			ELSE p.provider::text
		END
		FROM providers p
		WHERE m.provider_id = p.id;
	`)
	if err != nil {
		return err
	}

	// Make provider_type NOT NULL after migration
	_, err = tx.Exec(`
		ALTER TABLE models
		ALTER COLUMN provider_type SET NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Drop foreign key constraint on provider_id
	_, err = tx.Exec(`
		ALTER TABLE models
		DROP CONSTRAINT IF EXISTS models_provider_id_fkey;
	`)
	if err != nil {
		return err
	}

	// Drop provider_id column
	_, err = tx.Exec(`
		ALTER TABLE models
		DROP COLUMN IF EXISTS provider_id CASCADE;
	`)
	if err != nil {
		return err
	}

	// Drop provider_api_keys table
	_, err = tx.Exec(`
		DROP TABLE IF EXISTS provider_api_keys CASCADE;
	`)
	if err != nil {
		return err
	}

	// Drop providers table
	_, err = tx.Exec(`
		DROP TABLE IF EXISTS providers CASCADE;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251206113315_redesign_providers_to_api_keys_down(tx *sqlx.Tx) error {
	// Recreate providers table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS providers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			provider VARCHAR(50) NOT NULL,
			base_url VARCHAR(500),
			custom_headers JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Recreate provider_api_keys table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS provider_api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
			api_key TEXT NOT NULL,
			is_default BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Add provider_id column back to models
	_, err = tx.Exec(`
		ALTER TABLE models
		ADD COLUMN IF NOT EXISTS provider_id UUID;
	`)
	if err != nil {
		return err
	}

	// Migrate provider_type back to provider_id (this will require manual intervention)
	// We can't automatically map provider_type back to provider_id without data

	// Drop provider_type column
	_, err = tx.Exec(`
		ALTER TABLE models
		DROP COLUMN IF EXISTS provider_type CASCADE;
	`)
	if err != nil {
		return err
	}

	// Drop api_keys table
	_, err = tx.Exec(`
		DROP TABLE IF EXISTS api_keys CASCADE;
	`)
	if err != nil {
		return err
	}

	return nil
}
