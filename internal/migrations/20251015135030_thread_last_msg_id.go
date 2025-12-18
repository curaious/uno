package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251015135030",
		up:      mig_20251015135030_thread_last_msg_id_up,
		down:    mig_20251015135030_thread_last_msg_id_down,
	})
}

func mig_20251015135030_thread_last_msg_id_up(tx *sqlx.Tx) error {
	// Add last_message_id to threads table
	tx.MustExec(`
		ALTER TABLE threads
		ADD COLUMN last_message_id VARCHAR(255);
	`)

	// Create index on last_message_id for better performance
	tx.MustExec(`
		CREATE INDEX IF NOT EXISTS idx_threads_last_message_id ON threads(last_message_id);
	`)

	return nil
}

func mig_20251015135030_thread_last_msg_id_down(tx *sqlx.Tx) error {
	// Remove last_message_id from threads table
	tx.MustExec(`
		ALTER TABLE threads
		DROP COLUMN last_message_id;
	`)

	return nil
}
