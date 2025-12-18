package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251013083209",
		up:      mig_20251013083209_msg_history_up,
		down:    mig_20251013083209_msg_history_down,
	})
}

func mig_20251013083209_msg_history_up(tx *sqlx.Tx) error {
	// Create conversations table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			namespace_id VARCHAR(255) NOT NULL,
			conversation_id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL DEFAULT 'Untitled Conversation',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Create threads table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS threads (
			conversation_id VARCHAR(255) NOT NULL,
			origin_message_id VARCHAR(255),
			thread_id VARCHAR(255) PRIMARY KEY,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			FOREIGN KEY (conversation_id) REFERENCES conversations(conversation_id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create messages table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id VARCHAR(255) PRIMARY KEY,
			thread_id VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL,
			content JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			FOREIGN KEY (thread_id) REFERENCES threads(thread_id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes for better performance
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_conversations_namespace_id ON conversations(namespace_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_threads_conversation_id ON threads(conversation_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages(thread_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251013083209_msg_history_down(tx *sqlx.Tx) error {
	// Drop tables in reverse order due to foreign key constraints
	_, err := tx.Exec(`DROP TABLE IF EXISTS messages;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS threads;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS conversations;`)
	if err != nil {
		return err
	}

	return nil
}
