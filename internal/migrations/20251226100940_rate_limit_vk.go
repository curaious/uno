package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251226100940",
		up:      mig_20251226100940_rate_limit_vk_up,
		down:    mig_20251226100940_rate_limit_vk_down,
	})
}

func mig_20251226100940_rate_limit_vk_up(tx *sqlx.Tx) error {
	// Add rate_limits JSONB column to virtual_keys table
	_, err := tx.Exec(`
		ALTER TABLE virtual_keys
		ADD COLUMN IF NOT EXISTS rate_limits JSONB DEFAULT '[]'::jsonb;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251226100940_rate_limit_vk_down(tx *sqlx.Tx) error {
	// Remove rate_limits column from virtual_keys table
	_, err := tx.Exec(`
		ALTER TABLE virtual_keys
		DROP COLUMN IF EXISTS rate_limits;
	`)
	if err != nil {
		return err
	}

	return nil
}
