package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251102154653",
		up:      mig_20251102154653_create_agents_up,
		down:    mig_20251102154653_create_agents_down,
	})
}

func mig_20251102154653_create_agents_up(tx *sqlx.Tx) error {
	// Create agents table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			model_id UUID NOT NULL REFERENCES models(id) ON DELETE RESTRICT,
			prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE RESTRICT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, name)
		);
	`)
	if err != nil {
		return err
	}

	// Create agent_mcp_servers junction table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS agent_mcp_servers (
			agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			mcp_server_id UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
			tool_filters JSONB DEFAULT '[]'::jsonb,
			PRIMARY KEY (agent_id, mcp_server_id)
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agents_project_id ON agents(project_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agents_model_id ON agents(model_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agents_prompt_id ON agents(prompt_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_mcp_servers_agent_id ON agent_mcp_servers(agent_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_mcp_servers_mcp_server_id ON agent_mcp_servers(mcp_server_id);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251102154653_create_agents_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS agent_mcp_servers;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS agents;`)
	return err
}
