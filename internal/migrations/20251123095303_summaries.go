package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251123095303",
		up:      mig_20251123095303_summaries_up,
		down:    mig_20251123095303_summaries_down,
	})
}

func mig_20251123095303_summaries_up(tx *sqlx.Tx) error {
	// Create summaries table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS summaries (
			id VARCHAR(255) PRIMARY KEY,
			thread_id VARCHAR(255) NOT NULL,
			summary_message JSONB NOT NULL,
			last_summarized_message_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			meta JSONB,
			FOREIGN KEY (thread_id) REFERENCES threads(thread_id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes for efficient lookups
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_summaries_thread_id ON summaries(thread_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_summaries_last_summarized_message_id ON summaries(last_summarized_message_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_summaries_created_at ON summaries(created_at);
	`)
	if err != nil {
		return err
	}

	// Composite index for finding latest summary before a message
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_summaries_thread_created ON summaries(thread_id, created_at DESC);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251123095303_summaries_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS summaries;`)
	return err
}
