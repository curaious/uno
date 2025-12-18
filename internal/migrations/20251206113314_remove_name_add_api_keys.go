package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251206113314",
		up:      mig_20251206113314_remove_name_add_api_keys_up,
		down:    mig_20251206113314_remove_name_add_api_keys_down,
	})
}

func mig_20251206113314_remove_name_add_api_keys_up(tx *sqlx.Tx) error {
	// Create provider_api_keys table to store multiple API keys per provider
	_, err := tx.Exec(`
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

	// Create index on provider_id
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_provider_api_keys_provider_id 
		ON provider_api_keys(provider_id);
	`)
	if err != nil {
		return err
	}

	// Create unique index to ensure only one default key per provider
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_api_keys_provider_default 
		ON provider_api_keys(provider_id, is_default) 
		WHERE is_default = true;
	`)
	if err != nil {
		return err
	}

	// Migrate existing api_key data to provider_api_keys table
	_, err = tx.Exec(`
		INSERT INTO provider_api_keys (provider_id, api_key, is_default)
		SELECT id, api_key, true
		FROM providers
		WHERE api_key IS NOT NULL AND api_key != '';
	`)
	if err != nil {
		return err
	}

	// Drop the unique constraint on name
	_, err = tx.Exec(`
		DROP INDEX IF EXISTS idx_providers_name_unique;
	`)
	if err != nil {
		return err
	}

	// Drop the name column
	_, err = tx.Exec(`
		ALTER TABLE providers
		DROP COLUMN IF EXISTS name CASCADE;
	`)
	if err != nil {
		return err
	}

	// Drop the api_key column from providers (now stored in provider_api_keys)
	_, err = tx.Exec(`
		ALTER TABLE providers
		DROP COLUMN IF EXISTS api_key CASCADE;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251206113314_remove_name_add_api_keys_down(tx *sqlx.Tx) error {
	// Add name column back
	_, err := tx.Exec(`
		ALTER TABLE providers
		ADD COLUMN IF NOT EXISTS name VARCHAR(255);
	`)
	if err != nil {
		return err
	}

	// Add api_key column back
	_, err = tx.Exec(`
		ALTER TABLE providers
		ADD COLUMN IF NOT EXISTS api_key TEXT;
	`)
	if err != nil {
		return err
	}

	// Migrate default API key back to providers.api_key
	_, err = tx.Exec(`
		UPDATE providers p
		SET api_key = (
			SELECT api_key 
			FROM provider_api_keys pak 
			WHERE pak.provider_id = p.id 
			AND pak.is_default = true 
			LIMIT 1
		);
	`)
	if err != nil {
		return err
	}

	// Create unique constraint on name
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_name_unique 
		ON providers(name);
	`)
	if err != nil {
		return err
	}

	// Drop indexes from provider_api_keys
	_, err = tx.Exec(`
		DROP INDEX IF EXISTS idx_provider_api_keys_provider_default;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		DROP INDEX IF EXISTS idx_provider_api_keys_provider_id;
	`)
	if err != nil {
		return err
	}

	// Drop provider_api_keys table
	_, err = tx.Exec(`
		DROP TABLE IF EXISTS provider_api_keys;
	`)
	if err != nil {
		return err
	}

	return nil
}
