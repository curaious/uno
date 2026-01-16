package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20260116024327",
		up:      mig_20260116024327_agent_config_alias_up,
		down:    mig_20260116024327_agent_config_alias_down,
	})
}

func mig_20260116024327_agent_config_alias_up(tx *sqlx.Tx) error {
	// Create agent_config_aliases table for storing named mappings to agent versions
	// An alias can map to 1 or 2 versions, with an optional weight for the second version
	// Note: version validation is done at the application level since we can't create
	// a composite foreign key on (agent_id, version) directly
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS agent_config_aliases (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			agent_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			version1 INT NOT NULL,
			version2 INT,
			weight DECIMAL(5,2) CHECK (weight IS NULL OR (weight >= 0 AND weight <= 100)),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, agent_id, name),
			CHECK ((version2 IS NULL AND weight IS NULL) OR (version2 IS NOT NULL AND weight IS NOT NULL))
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_config_aliases_project_id ON agent_config_aliases(project_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_config_aliases_agent_id ON agent_config_aliases(agent_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_config_aliases_name ON agent_config_aliases(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_config_aliases_project_agent ON agent_config_aliases(project_id, agent_id);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20260116024327_agent_config_alias_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS agent_config_aliases;`)
	return err
}
