package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20260120054454",
		up:      mig_20260120054454_weight_type_up,
		down:    mig_20260120054454_weight_type_down,
	})
}

func mig_20260120054454_weight_type_up(tx *sqlx.Tx) error {
	// Drop the existing CHECK constraint on the weight column
	_, err := tx.Exec(`
		ALTER TABLE agent_config_aliases
		DROP CONSTRAINT IF EXISTS agent_config_aliases_weight_check;
	`)
	if err != nil {
		return err
	}

	// Alter the weight column from DECIMAL(5,2) to INT
	// Using ROUND to convert decimal values to integers
	_, err = tx.Exec(`
		ALTER TABLE agent_config_aliases
		ALTER COLUMN weight TYPE INT
		USING CASE 
			WHEN weight IS NULL THEN NULL
			ELSE ROUND(weight)::INT
		END;
	`)
	if err != nil {
		return err
	}

	// Re-add the CHECK constraint for INT type
	_, err = tx.Exec(`
		ALTER TABLE agent_config_aliases
		ADD CONSTRAINT agent_config_aliases_weight_check 
		CHECK (weight IS NULL OR (weight >= 0 AND weight <= 100));
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20260120054454_weight_type_down(tx *sqlx.Tx) error {
	// Revert the weight column from INT back to DECIMAL(5,2)
	_, err := tx.Exec(`
		ALTER TABLE agent_config_aliases
		ALTER COLUMN weight TYPE DECIMAL(5,2)
		USING CASE 
			WHEN weight IS NULL THEN NULL
			ELSE weight::DECIMAL(5,2)
		END;
	`)
	if err != nil {
		return err
	}

	// Drop the constraint
	_, err = tx.Exec(`
		ALTER TABLE agent_config_aliases
		DROP CONSTRAINT IF EXISTS agent_config_aliases_weight_check;
	`)
	if err != nil {
		return err
	}

	// Re-add the CHECK constraint for DECIMAL type
	_, err = tx.Exec(`
		ALTER TABLE agent_config_aliases
		ADD CONSTRAINT agent_config_aliases_weight_check 
		CHECK (weight IS NULL OR (weight >= 0 AND weight <= 100));
	`)
	if err != nil {
		return err
	}

	return nil
}
