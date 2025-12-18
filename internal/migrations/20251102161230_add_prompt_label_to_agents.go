package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251102161230",
		up:      mig_20251102161230_add_prompt_label_to_agents_up,
		down:    mig_20251102161230_add_prompt_label_to_agents_down,
	})
}

func mig_20251102161230_add_prompt_label_to_agents_up(tx *sqlx.Tx) error {
	// Add prompt_label column to agents table with default value "latest"
	_, err := tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS prompt_label VARCHAR(50) DEFAULT 'latest';
	`)
	if err != nil {
		return err
	}

	// Update existing rows to have "latest" as default if null
	_, err = tx.Exec(`
		UPDATE agents 
		SET prompt_label = 'latest' 
		WHERE prompt_label IS NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251102161230_add_prompt_label_to_agents_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS prompt_label;
	`)
	return err
}
