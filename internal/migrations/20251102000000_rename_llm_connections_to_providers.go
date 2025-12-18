package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251102000000",
		up:      mig_20251102000000_rename_llm_connections_to_providers_up,
		down:    mig_20251102000000_rename_llm_connections_to_providers_down,
	})
}

func mig_20251102000000_rename_llm_connections_to_providers_up(tx *sqlx.Tx) error {
	// Rename table from llm_connections to providers
	_, err := tx.Exec(`
		ALTER TABLE llm_connections RENAME TO providers;
	`)
	if err != nil {
		return err
	}

	// Rename indexes
	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_llm_connections_provider RENAME TO idx_providers_provider;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_llm_connections_name RENAME TO idx_providers_name;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_llm_connections_project_id RENAME TO idx_providers_project_id;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_llm_connections_project_name RENAME TO idx_providers_project_name;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251102000000_rename_llm_connections_to_providers_down(tx *sqlx.Tx) error {
	// Rename indexes back
	_, err := tx.Exec(`
		ALTER INDEX IF EXISTS idx_providers_project_name RENAME TO idx_llm_connections_project_name;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_providers_project_id RENAME TO idx_llm_connections_project_id;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_providers_name RENAME TO idx_llm_connections_name;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER INDEX IF EXISTS idx_providers_provider RENAME TO idx_llm_connections_provider;
	`)
	if err != nil {
		return err
	}

	// Rename table back
	_, err = tx.Exec(`
		ALTER TABLE providers RENAME TO llm_connections;
	`)
	if err != nil {
		return err
	}

	return nil
}
