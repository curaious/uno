package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20260115012334",
		up:      mig_20260115012334_agent_config_up,
		down:    mig_20260115012334_agent_config_down,
	})
}

func mig_20260115012334_agent_config_up(tx *sqlx.Tx) error {
	// Create agent_configs table for storing complete agent configurations as JSON
	// This table stores versioned agent configurations including model, prompt, schema,
	// MCP servers, and history settings
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS agent_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			version INT NOT NULL DEFAULT 1,
			config JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, name, version)
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_configs_project_id ON agent_configs(project_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_configs_name ON agent_configs(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_configs_project_name ON agent_configs(project_id, name);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20260115012334_agent_config_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS agent_configs;`)
	return err
}
