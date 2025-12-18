package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251016011750",
		up:      mig_20251016011750_message_usage_up,
		down:    mig_20251016011750_message_usage_down,
	})
}

func mig_20251016011750_message_usage_up(tx *sqlx.Tx) error {
	// Add meta JSONB column to messages table
	tx.MustExec(`
		ALTER TABLE messages
		ADD COLUMN meta JSONB DEFAULT '{}';
	`)

	return nil
}

func mig_20251016011750_message_usage_down(tx *sqlx.Tx) error {
	// Remove meta column from messages table
	tx.MustExec(`
		ALTER TABLE messages
		DROP COLUMN meta;
	`)

	return nil
}
