package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20260116021725",
		up:      mig_20260116021725_agent_config_version_up,
		down:    mig_20260116021725_agent_config_version_down,
	})
}

func mig_20260116021725_agent_config_version_up(tx *sqlx.Tx) error {
	// Add agent_id column (stable UUID per agent across versions)
	_, err := tx.Exec(`
		ALTER TABLE agent_configs 
		ADD COLUMN IF NOT EXISTS agent_id UUID;
	`)
	if err != nil {
		return err
	}

	// For existing rows, set agent_id = id (each existing row gets its own agent_id)
	_, err = tx.Exec(`
		UPDATE agent_configs 
		SET agent_id = id 
		WHERE agent_id IS NULL;
	`)
	if err != nil {
		return err
	}

	// Make agent_id NOT NULL
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		ALTER COLUMN agent_id SET NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Add immutable column (true for versions > 0, false for version 0)
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		ADD COLUMN IF NOT EXISTS immutable BOOLEAN NOT NULL DEFAULT false;
	`)
	if err != nil {
		return err
	}

	// Set immutable = true for all existing versions > 0
	// First, update existing version 1 rows to version 0 (since default should be 0)
	_, err = tx.Exec(`
		UPDATE agent_configs 
		SET version = 0, immutable = false
		WHERE version = 1;
	`)
	if err != nil {
		return err
	}

	// Set immutable = true for any versions > 0 (if any exist)
	_, err = tx.Exec(`
		UPDATE agent_configs 
		SET immutable = true
		WHERE version > 0;
	`)
	if err != nil {
		return err
	}

	// Change default version to 0
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		ALTER COLUMN version SET DEFAULT 0;
	`)
	if err != nil {
		return err
	}

	// Drop old unique constraint on (project_id, name, version)
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		DROP CONSTRAINT IF EXISTS agent_configs_project_id_name_version_key;
	`)
	if err != nil {
		return err
	}

	// Add new unique constraint on (agent_id, version)
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		ADD CONSTRAINT agent_configs_agent_id_version_key UNIQUE (agent_id, version);
	`)
	if err != nil {
		return err
	}

	// Create index on agent_id for faster lookups
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_configs_agent_id ON agent_configs(agent_id);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20260116021725_agent_config_version_down(tx *sqlx.Tx) error {
	// Drop the new unique constraint
	_, err := tx.Exec(`
		ALTER TABLE agent_configs 
		DROP CONSTRAINT IF EXISTS agent_configs_agent_id_version_key;
	`)
	if err != nil {
		return err
	}

	// Restore old unique constraint
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		ADD CONSTRAINT agent_configs_project_id_name_version_key UNIQUE (project_id, name, version);
	`)
	if err != nil {
		return err
	}

	// Drop index
	_, err = tx.Exec(`
		DROP INDEX IF EXISTS idx_agent_configs_agent_id;
	`)
	if err != nil {
		return err
	}

	// Change default version back to 1
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		ALTER COLUMN version SET DEFAULT 1;
	`)
	if err != nil {
		return err
	}

	// Restore version 0 rows back to version 1
	_, err = tx.Exec(`
		UPDATE agent_configs 
		SET version = 1
		WHERE version = 0;
	`)
	if err != nil {
		return err
	}

	// Drop immutable column
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		DROP COLUMN IF EXISTS immutable;
	`)
	if err != nil {
		return err
	}

	// Drop agent_id column
	_, err = tx.Exec(`
		ALTER TABLE agent_configs 
		DROP COLUMN IF EXISTS agent_id;
	`)
	if err != nil {
		return err
	}

	return nil
}
