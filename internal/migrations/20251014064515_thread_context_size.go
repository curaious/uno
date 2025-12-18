package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251014064515",
		up:      mig_20251014064515_thread_context_size_up,
		down:    mig_20251014064515_thread_context_size_down,
	})
}

func mig_20251014064515_thread_context_size_up(tx *sqlx.Tx) error {
	// Add meta column to threads table for storing JSON data
	_, err := tx.Exec(`
		ALTER TABLE threads 
		ADD COLUMN meta JSONB DEFAULT '{}'::jsonb;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251014064515_thread_context_size_down(tx *sqlx.Tx) error {
	// Remove the meta column from threads table
	_, err := tx.Exec(`ALTER TABLE threads DROP COLUMN IF EXISTS meta;`)
	if err != nil {
		return err
	}

	return nil
}
