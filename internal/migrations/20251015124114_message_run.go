package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251015124114",
		up:      mig_20251015124114_message_run_up,
		down:    mig_20251015124114_message_run_down,
	})
}

func mig_20251015124114_message_run_up(tx *sqlx.Tx) error {
	// add "run_id" and "conversation_id" and "messages" to message table
	_, err := tx.Exec(`
		ALTER TABLE messages
		ADD COLUMN conversation_id VARCHAR(255),
		ADD COLUMN messages JSONB;
	`)
	if err != nil {
		return err
	}

	// Drop role column from message table
	_, err = tx.Exec(`
		ALTER TABLE messages
		DROP COLUMN role,
	 	DROP COLUMN content;
	`)
	return nil
}

func mig_20251015124114_message_run_down(tx *sqlx.Tx) error {
	// remove "run_id" and "conversation_id" and "messages" from message table
	_, err := tx.Exec(`
		ALTER TABLE messages
		DROP COLUMN conversation_id,
		DROP COLUMN messages;
	`)
	if err != nil {
		return err
	}

	// add role column to message table
	_, err = tx.Exec(`
		ALTER TABLE messages
		ADD COLUMN role VARCHAR(50) NOT NULL,
		ADD COLUMN content JSONB NOT NULL;
	`)
	if err != nil {
		return err
	}
	return nil
}
