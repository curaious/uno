package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251217092314",
		up:      mig_20251217092314_schemas_up,
		down:    mig_20251217092314_schemas_down,
	})
}

func mig_20251217092314_schemas_up(tx *sqlx.Tx) error {
	// Create schemas table for storing JSON schemas
	// source_type is designed for future support of golang struct / typescript interface conversion
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS schemas (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			schema JSONB NOT NULL,
			source_type VARCHAR(50) DEFAULT 'manual',
			source_content TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, name)
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_schemas_project_id ON schemas(project_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_schemas_name ON schemas(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_schemas_source_type ON schemas(source_type);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251217092314_schemas_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS schemas;`)
	return err
}
