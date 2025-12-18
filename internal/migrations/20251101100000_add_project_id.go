package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251101100000",
		up:      mig_20251101100000_add_project_id_up,
		down:    mig_20251101100000_add_project_id_down,
	})
}

func mig_20251101100000_add_project_id_up(tx *sqlx.Tx) error {
	// Add project_id to llm_connections
	_, err := tx.Exec(`
		ALTER TABLE llm_connections
		ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_llm_connections_project_id ON llm_connections(project_id);
	`)
	if err != nil {
		return err
	}

	// Update unique constraint on llm_connections to include project_id
	_, err = tx.Exec(`
		ALTER TABLE llm_connections
		DROP CONSTRAINT IF EXISTS llm_connections_name_key;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_connections_project_name 
		ON llm_connections(project_id, name) WHERE project_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Add project_id to mcp_servers
	_, err = tx.Exec(`
		ALTER TABLE mcp_servers
		ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_mcp_servers_project_id ON mcp_servers(project_id);
	`)
	if err != nil {
		return err
	}

	// Update unique constraint on mcp_servers to include project_id
	_, err = tx.Exec(`
		ALTER TABLE mcp_servers
		DROP CONSTRAINT IF EXISTS mcp_servers_name_key;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_mcp_servers_project_name 
		ON mcp_servers(project_id, name) WHERE project_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Replace namespace_id with project_id in conversations
	// First, check if project_id column already exists
	_, err = tx.Exec(`
		ALTER TABLE conversations
		ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_conversations_project_id ON conversations(project_id);
	`)
	if err != nil {
		return err
	}

	// Migrate data: if namespace_id exists and we want to keep both temporarily
	// For now, we'll keep both columns and let the application layer handle the migration

	// Add project_id to prompts
	_, err = tx.Exec(`
		ALTER TABLE prompts
		ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_prompts_project_id ON prompts(project_id);
	`)
	if err != nil {
		return err
	}

	// Update unique constraint on prompts to include project_id
	_, err = tx.Exec(`
		ALTER TABLE prompts
		DROP CONSTRAINT IF EXISTS prompts_name_key;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_prompts_project_name 
		ON prompts(project_id, name) WHERE project_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251101100000_add_project_id_down(tx *sqlx.Tx) error {
	// Remove indexes
	_, err := tx.Exec(`DROP INDEX IF EXISTS idx_llm_connections_project_id;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP INDEX IF EXISTS idx_llm_connections_project_name;`)
	if err != nil {
		return err
	}

	// Restore unique constraint on name
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS llm_connections_name_key ON llm_connections(name);
	`)
	if err != nil {
		return err
	}

	// Remove project_id from llm_connections
	_, err = tx.Exec(`ALTER TABLE llm_connections DROP COLUMN IF EXISTS project_id;`)
	if err != nil {
		return err
	}

	// Remove indexes from mcp_servers
	_, err = tx.Exec(`DROP INDEX IF EXISTS idx_mcp_servers_project_id;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP INDEX IF EXISTS idx_mcp_servers_project_name;`)
	if err != nil {
		return err
	}

	// Restore unique constraint on name
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS mcp_servers_name_key ON mcp_servers(name);
	`)
	if err != nil {
		return err
	}

	// Remove project_id from mcp_servers
	_, err = tx.Exec(`ALTER TABLE mcp_servers DROP COLUMN IF EXISTS project_id;`)
	if err != nil {
		return err
	}

	// Remove index from conversations
	_, err = tx.Exec(`DROP INDEX IF EXISTS idx_conversations_project_id;`)
	if err != nil {
		return err
	}

	// Remove project_id from conversations
	_, err = tx.Exec(`ALTER TABLE conversations DROP COLUMN IF EXISTS project_id;`)
	if err != nil {
		return err
	}

	// Remove indexes from prompts
	_, err = tx.Exec(`DROP INDEX IF EXISTS idx_prompts_project_id;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP INDEX IF EXISTS idx_prompts_project_name;`)
	if err != nil {
		return err
	}

	// Restore unique constraint on name
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS prompts_name_key ON prompts(name);
	`)
	if err != nil {
		return err
	}

	// Remove project_id from prompts
	_, err = tx.Exec(`ALTER TABLE prompts DROP COLUMN IF EXISTS project_id;`)
	if err != nil {
		return err
	}

	return nil
}
