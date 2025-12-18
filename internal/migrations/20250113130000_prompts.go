package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20250113130000",
		up:      mig_20250113130000_prompts_up,
		down:    mig_20250113130000_prompts_down,
	})
}

func mig_20250113130000_prompts_up(tx *sqlx.Tx) error {
	// Create prompts table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS prompts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Create prompt_versions table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS prompt_versions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
			version INTEGER NOT NULL,
			template TEXT NOT NULL,
			commit_message TEXT NOT NULL,
			label VARCHAR(50) CHECK (label IN ('production', 'latest')),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(prompt_id, version),
			UNIQUE(prompt_id, label)
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_prompts_name ON prompts(name);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_prompt_versions_prompt_id ON prompt_versions(prompt_id);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_prompt_versions_label ON prompt_versions(label);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_prompt_versions_version ON prompt_versions(prompt_id, version);
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20250113130000_prompts_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS prompt_versions;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS prompts;`)
	return err
}
