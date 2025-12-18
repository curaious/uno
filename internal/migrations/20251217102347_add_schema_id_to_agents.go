package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251217102347",
		up:      mig_20251217102347_add_schema_id_to_agents_up,
		down:    mig_20251217102347_add_schema_id_to_agents_down,
	})
}

func mig_20251217102347_add_schema_id_to_agents_up(tx *sqlx.Tx) error {
	// Add optional schema_id column to agents table
	_, err := tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS schema_id UUID REFERENCES schemas(id) ON DELETE SET NULL;
	`)
	if err != nil {
		return err
	}

	// Create index for schema_id
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agents_schema_id ON agents(schema_id);
	`)
	return err
}

func mig_20251217102347_add_schema_id_to_agents_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`ALTER TABLE agents DROP COLUMN IF EXISTS schema_id;`)
	return err
}
