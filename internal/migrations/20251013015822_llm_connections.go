package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251013015822",
		up:      mig_20251013015822_llm_connections_up,
		down:    mig_20251013015822_llm_connections_down,
	})
}

func mig_20251013015822_llm_connections_up(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS llm_connections (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			provider VARCHAR(50) NOT NULL CHECK (provider IN ('openai', 'anthropic', 'google', 'xai')),
			api_key TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(name)
		);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_llm_connections_provider ON llm_connections(provider);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_llm_connections_name ON llm_connections(name);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251013015822_llm_connections_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS llm_connections;`)
	return err
}
