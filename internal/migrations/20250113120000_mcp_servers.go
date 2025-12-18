package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20250113120000",
		up:      mig_20250113120000_mcp_servers_up,
		down:    mig_20250113120000_mcp_servers_down,
	})
}

func mig_20250113120000_mcp_servers_up(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS mcp_servers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			endpoint VARCHAR(500) NOT NULL,
			headers JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(name)
		);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_mcp_servers_name ON mcp_servers(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_mcp_servers_endpoint ON mcp_servers(endpoint);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20250113120000_mcp_servers_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS mcp_servers;`)
	return err
}
