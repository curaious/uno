package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251206120000",
		up:      mig_20251206120000_create_virtual_keys_up,
		down:    mig_20251206120000_create_virtual_keys_down,
	})
}

func mig_20251206120000_create_virtual_keys_up(tx *sqlx.Tx) error {
	// Create virtual_keys table (global, no project_id)
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS virtual_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Create virtual_key_providers table (many-to-many relationship)
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS virtual_key_providers (
			virtual_key_id UUID NOT NULL REFERENCES virtual_keys(id) ON DELETE CASCADE,
			provider_type VARCHAR(50) NOT NULL,
			PRIMARY KEY (virtual_key_id, provider_type)
		);
	`)
	if err != nil {
		return err
	}

	// Create virtual_key_models table (many-to-many relationship with models)
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS virtual_key_models (
			virtual_key_id UUID NOT NULL REFERENCES virtual_keys(id) ON DELETE CASCADE,
			model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
			PRIMARY KEY (virtual_key_id, model_id)
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_keys_name ON virtual_keys(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_key_providers_virtual_key_id ON virtual_key_providers(virtual_key_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_key_providers_provider_type ON virtual_key_providers(provider_type);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_key_models_virtual_key_id ON virtual_key_models(virtual_key_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_key_models_model_id ON virtual_key_models(model_id);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251206120000_create_virtual_keys_down(tx *sqlx.Tx) error {
	// Drop tables in reverse order (due to foreign keys)
	_, err := tx.Exec(`DROP TABLE IF EXISTS virtual_key_models CASCADE;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS virtual_key_providers CASCADE;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS virtual_keys CASCADE;`)
	if err != nil {
		return err
	}

	return nil
}
