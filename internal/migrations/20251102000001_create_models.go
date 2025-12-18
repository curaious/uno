package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251102000001",
		up:      mig_20251102000001_create_models_up,
		down:    mig_20251102000001_create_models_down,
	})
}

func mig_20251102000001_create_models_up(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS models (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			model_id VARCHAR(255) NOT NULL,
			parameters JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, name)
		);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_models_project_id ON models(project_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_models_provider_id ON models(provider_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_models_name ON models(name);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251102000001_create_models_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS models;`)
	return err
}
