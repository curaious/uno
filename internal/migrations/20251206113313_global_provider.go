package migrations

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	m.addMigration(&migration{
		version: "20251206113313",
		up:      mig_20251206113313_global_provider_up,
		down:    mig_20251206113313_global_provider_down,
	})
}

func mig_20251206113313_global_provider_up(tx *sqlx.Tx) error {
	// Drop the unique constraint on (project_id, name)
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_providers_project_name;
	`)
	if err != nil {
		return err
	}

	// Drop the foreign key constraint on project_id
	// Try common constraint name patterns
	constraintNames := []string{
		"providers_project_id_fkey",
		"llm_connections_project_id_fkey", // In case it wasn't renamed
	}

	for _, constraintName := range constraintNames {
		_, err = tx.Exec(fmt.Sprintf(`
			ALTER TABLE providers
			DROP CONSTRAINT IF EXISTS %s;
		`, constraintName))
		// Continue even if constraint doesn't exist
		if err != nil {
			// Log but continue - constraint might not exist or have different name
		}
	}

	// Drop the index on project_id
	_, err = tx.Exec(`
		DROP INDEX IF EXISTS idx_providers_project_id;
	`)
	if err != nil {
		return err
	}

	// Drop the project_id column (CASCADE will drop any remaining constraints)
	_, err = tx.Exec(`
		ALTER TABLE providers
		DROP COLUMN IF EXISTS project_id CASCADE;
	`)
	if err != nil {
		return err
	}

	// Create a unique constraint on name (globally)
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_name_unique 
		ON providers(name);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251206113313_global_provider_down(tx *sqlx.Tx) error {
	// Drop the unique constraint on name
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_providers_name_unique;
	`)
	if err != nil {
		return err
	}

	// Add project_id column back (nullable initially, as we don't have historical data)
	_, err = tx.Exec(`
		ALTER TABLE providers
		ADD COLUMN IF NOT EXISTS project_id UUID;
	`)
	if err != nil {
		return err
	}

	// Create index on project_id
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_providers_project_id ON providers(project_id);
	`)
	if err != nil {
		return err
	}

	// Add foreign key constraint (deferrable to allow NULL values initially)
	_, err = tx.Exec(`
		ALTER TABLE providers
		ADD CONSTRAINT providers_project_id_fkey 
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;
	`)
	if err != nil {
		return err
	}

	// Create unique constraint on (project_id, name)
	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_project_name 
		ON providers(project_id, name) WHERE project_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}
