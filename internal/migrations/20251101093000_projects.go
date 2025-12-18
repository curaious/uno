package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251101093000",
		up:      mig_20251101093000_projects_up,
		down:    mig_20251101093000_projects_down,
	})
}

func mig_20251101093000_projects_up(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
        CREATE TABLE IF NOT EXISTS projects (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(name)
        );
    `)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
        CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
    `)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251101093000_projects_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS projects;`)
	return err
}
