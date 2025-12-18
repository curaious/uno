package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251206121000",
		up:      mig_20251206121000_add_secret_key_to_virtual_keys_up,
		down:    mig_20251206121000_add_secret_key_to_virtual_keys_down,
	})
}

func mig_20251206121000_add_secret_key_to_virtual_keys_up(tx *sqlx.Tx) error {
	// Add secret_key column to virtual_keys table
	_, err := tx.Exec(`
		ALTER TABLE virtual_keys
		ADD COLUMN IF NOT EXISTS secret_key VARCHAR(255) NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	// Create unique index on secret_key
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_virtual_keys_secret_key ON virtual_keys(secret_key);
	`)
	if err != nil {
		return err
	}

	// Generate secret keys for existing virtual keys (if any)
	// This will be handled by the application layer when creating new keys
	// For existing keys, we'll leave them empty and they'll need to be regenerated

	return nil
}

func mig_20251206121000_add_secret_key_to_virtual_keys_down(tx *sqlx.Tx) error {
	// Drop the index first
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_virtual_keys_secret_key;
	`)
	if err != nil {
		return err
	}

	// Drop the column
	_, err = tx.Exec(`
		ALTER TABLE virtual_keys
		DROP COLUMN IF EXISTS secret_key;
	`)
	if err != nil {
		return err
	}

	return nil
}
