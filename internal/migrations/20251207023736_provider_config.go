package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251207023736",
		up:      mig_20251207023736_provider_config_up,
		down:    mig_20251207023736_provider_config_down,
	})
}

func mig_20251207023736_provider_config_up(tx *sqlx.Tx) error {
	// Create provider_configs table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS provider_configs (
			provider_type VARCHAR(50) PRIMARY KEY,
			base_url VARCHAR(500),
			custom_headers JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Migrate base_url and custom_headers from api_keys to provider_configs
	// For each provider type, use the default key's config, or the first enabled key's config
	_, err = tx.Exec(`
		INSERT INTO provider_configs (provider_type, base_url, custom_headers)
		SELECT DISTINCT ON (provider_type)
			provider_type,
			base_url,
			custom_headers
		FROM api_keys
		WHERE (base_url IS NOT NULL OR (custom_headers IS NOT NULL AND custom_headers != '{}'::jsonb))
		ORDER BY provider_type, is_default DESC, enabled DESC, created_at ASC
		ON CONFLICT (provider_type) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// Remove base_url and custom_headers columns from api_keys table
	_, err = tx.Exec(`
		ALTER TABLE api_keys
		DROP COLUMN IF EXISTS base_url,
		DROP COLUMN IF EXISTS custom_headers;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251207023736_provider_config_down(tx *sqlx.Tx) error {
	// Add back base_url and custom_headers columns to api_keys
	_, err := tx.Exec(`
		ALTER TABLE api_keys
		ADD COLUMN IF NOT EXISTS base_url VARCHAR(500),
		ADD COLUMN IF NOT EXISTS custom_headers JSONB DEFAULT '{}'::jsonb;
	`)
	if err != nil {
		return err
	}

	// Migrate data back from provider_configs to api_keys
	// Set base_url and custom_headers for all keys of each provider type
	_, err = tx.Exec(`
		UPDATE api_keys ak
		SET 
			base_url = pc.base_url,
			custom_headers = pc.custom_headers
		FROM provider_configs pc
		WHERE ak.provider_type = pc.provider_type;
	`)
	if err != nil {
		return err
	}

	// Drop provider_configs table
	_, err = tx.Exec(`DROP TABLE IF EXISTS provider_configs;`)
	if err != nil {
		return err
	}

	return nil
}
