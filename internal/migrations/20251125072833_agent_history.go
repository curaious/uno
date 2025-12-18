package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251125072833",
		up:      mig_20251125072833_agent_history_up,
		down:    mig_20251125072833_agent_history_down,
	})
}

func mig_20251125072833_agent_history_up(tx *sqlx.Tx) error {
	// Add enable_history column to agents table with default FALSE
	_, err := tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS enable_history BOOLEAN DEFAULT FALSE;
	`)
	if err != nil {
		return err
	}

	// Set enable_history to TRUE for existing agents
	_, err = tx.Exec(`
		UPDATE agents 
		SET enable_history = TRUE 
		WHERE enable_history IS NULL OR enable_history = FALSE;
	`)
	if err != nil {
		return err
	}

	// Add summarizer_type column (nullable, values: 'llm', 'sliding_window')
	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS summarizer_type VARCHAR(50);
	`)
	if err != nil {
		return err
	}

	// Add LLM summarizer specific columns
	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS llm_summarizer_token_threshold INTEGER;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS llm_summarizer_keep_recent_count INTEGER;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS llm_summarizer_prompt_id UUID REFERENCES prompts(id) ON DELETE RESTRICT;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS llm_summarizer_prompt_label VARCHAR(50);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS llm_summarizer_model_id UUID REFERENCES models(id) ON DELETE RESTRICT;
	`)
	if err != nil {
		return err
	}

	// Add sliding window summarizer specific column
	_, err = tx.Exec(`
		ALTER TABLE agents 
		ADD COLUMN IF NOT EXISTS sliding_window_keep_count INTEGER;
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251125072833_agent_history_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS sliding_window_keep_count;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS llm_summarizer_model_id;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS llm_summarizer_prompt_label;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS llm_summarizer_prompt_id;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS llm_summarizer_keep_recent_count;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS llm_summarizer_token_threshold;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS summarizer_type;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE agents 
		DROP COLUMN IF EXISTS enable_history;
	`)
	return err
}
