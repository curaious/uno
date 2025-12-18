package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251103091530",
		up:      mig_20251103091530_add_base_url_and_custom_headers_to_providers_up,
		down:    mig_20251103091530_add_base_url_and_custom_headers_to_providers_down,
	})
}

func mig_20251103091530_add_base_url_and_custom_headers_to_providers_up(tx *sqlx.Tx) error {
	// Add base_url column to providers table
	_, err := tx.Exec(`
		ALTER TABLE providers 
		ADD COLUMN IF NOT EXISTS base_url VARCHAR(500);
	`)
	if err != nil {
		return err
	}

	// Add custom_headers column to providers table as JSONB
	_, err = tx.Exec(`
		ALTER TABLE providers 
		ADD COLUMN IF NOT EXISTS custom_headers JSONB DEFAULT '{}'::jsonb;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251103091530_add_base_url_and_custom_headers_to_providers_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE providers 
		DROP COLUMN IF EXISTS custom_headers;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE providers 
		DROP COLUMN IF EXISTS base_url;
	`)
	return err
}
