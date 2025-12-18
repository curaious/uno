package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251208113251",
		up:      mig_20251208113251_key_for_project_up,
		down:    mig_20251208113251_key_for_project_down,
	})
}

func mig_20251208113251_key_for_project_up(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE projects 
		ADD COLUMN default_key TEXT;
	`)
	return err
}

func mig_20251208113251_key_for_project_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE projects 
		DROP COLUMN IF EXISTS default_key;
	`)
	return err
}
