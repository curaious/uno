package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251206122000",
		up:      mig_20251206122000_change_virtual_key_models_to_names_up,
		down:    mig_20251206122000_change_virtual_key_models_to_names_down,
	})
}

func mig_20251206122000_change_virtual_key_models_to_names_up(tx *sqlx.Tx) error {
	// Drop the old virtual_key_models table
	_, err := tx.Exec(`
		DROP TABLE IF EXISTS virtual_key_models CASCADE;
	`)
	if err != nil {
		return err
	}

	// Create new virtual_key_models table with model_name instead of model_id
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS virtual_key_models (
			virtual_key_id UUID NOT NULL REFERENCES virtual_keys(id) ON DELETE CASCADE,
			model_name VARCHAR(255) NOT NULL,
			PRIMARY KEY (virtual_key_id, model_name)
		);
	`)
	if err != nil {
		return err
	}

	// Create index
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_key_models_virtual_key_id ON virtual_key_models(virtual_key_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_virtual_key_models_model_name ON virtual_key_models(model_name);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251206122000_change_virtual_key_models_to_names_down(tx *sqlx.Tx) error {
	// Drop the new table
	_, err := tx.Exec(`
		DROP TABLE IF EXISTS virtual_key_models CASCADE;
	`)
	if err != nil {
		return err
	}

	// Recreate the old table with model_id
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

	// Recreate indexes
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
