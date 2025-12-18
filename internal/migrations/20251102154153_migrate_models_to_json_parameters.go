package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251102154153",
		up:      mig_20251102154153_migrate_models_to_json_parameters_up,
		down:    mig_20251102154153_migrate_models_to_json_parameters_down,
	})
}

func mig_20251102154153_migrate_models_to_json_parameters_up(tx *sqlx.Tx) error {
	// Check if models table exists
	var tableExists bool
	err := tx.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'models'
		);
	`)
	if err != nil {
		return err
	}

	if !tableExists {
		// Table doesn't exist yet, nothing to migrate
		return nil
	}

	// Add parameters column if it doesn't exist
	_, err = tx.Exec(`
		ALTER TABLE models 
		ADD COLUMN IF NOT EXISTS parameters JSONB DEFAULT '{}'::jsonb;
	`)
	if err != nil {
		return err
	}

	// Check if old columns exist
	var hasTemperature bool
	err = tx.Get(&hasTemperature, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'models' AND column_name = 'temperature'
		);
	`)
	if err != nil {
		return err
	}

	// If old columns exist, migrate data
	if hasTemperature {
		// Migrate existing data to JSONB parameters column
		// Only migrate if parameters is empty or default, and at least one old column has data
		_, err = tx.Exec(`
			UPDATE models 
			SET parameters = jsonb_build_object(
				'temperature', CASE WHEN temperature IS NOT NULL THEN temperature::text END,
				'max_tokens', CASE WHEN max_tokens IS NOT NULL THEN max_tokens::text END,
				'top_p', CASE WHEN top_p IS NOT NULL THEN top_p::text END,
				'frequency_penalty', CASE WHEN frequency_penalty IS NOT NULL THEN frequency_penalty::text END,
				'presence_penalty', CASE WHEN presence_penalty IS NOT NULL THEN presence_penalty::text END
			)
			WHERE (parameters IS NULL OR parameters = '{}'::jsonb) 
				AND (temperature IS NOT NULL OR max_tokens IS NOT NULL OR top_p IS NOT NULL 
					OR frequency_penalty IS NOT NULL OR presence_penalty IS NOT NULL);
		`)
		if err != nil {
			return err
		}

		// Remove NULL values from JSONB for all rows
		_, err = tx.Exec(`
			UPDATE models 
			SET parameters = (
				SELECT COALESCE(jsonb_object_agg(key, value), '{}'::jsonb)
				FROM jsonb_each(parameters)
				WHERE value::text != 'null'
			)
			WHERE parameters IS NOT NULL AND jsonb_typeof(parameters) = 'object';
		`)
		if err != nil {
			return err
		}

		// Drop old columns
		_, err = tx.Exec(`
			ALTER TABLE models DROP COLUMN IF EXISTS temperature;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models DROP COLUMN IF EXISTS max_tokens;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models DROP COLUMN IF EXISTS top_p;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models DROP COLUMN IF EXISTS frequency_penalty;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models DROP COLUMN IF EXISTS presence_penalty;
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

func mig_20251102154153_migrate_models_to_json_parameters_down(tx *sqlx.Tx) error {
	// Check if models table exists
	var tableExists bool
	err := tx.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'models'
		);
	`)
	if err != nil {
		return err
	}

	if !tableExists {
		return nil
	}

	// Check if old columns exist
	var hasTemperature bool
	err = tx.Get(&hasTemperature, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'models' AND column_name = 'temperature'
		);
	`)
	if err != nil {
		return err
	}

	// If old columns don't exist, add them back and migrate data
	if !hasTemperature {
		// Add back old columns
		_, err = tx.Exec(`
			ALTER TABLE models ADD COLUMN IF NOT EXISTS temperature DOUBLE PRECISION;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models ADD COLUMN IF NOT EXISTS max_tokens INTEGER;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models ADD COLUMN IF NOT EXISTS top_p DOUBLE PRECISION;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models ADD COLUMN IF NOT EXISTS frequency_penalty DOUBLE PRECISION;
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			ALTER TABLE models ADD COLUMN IF NOT EXISTS presence_penalty DOUBLE PRECISION;
		`)
		if err != nil {
			return err
		}

		// Migrate data back from JSONB
		_, err = tx.Exec(`
			UPDATE models 
			SET 
				temperature = CASE 
					WHEN parameters->>'temperature' IS NOT NULL AND parameters->>'temperature' != 'null'
					THEN (parameters->>'temperature')::double precision 
					ELSE NULL 
				END,
				max_tokens = CASE 
					WHEN parameters->>'max_tokens' IS NOT NULL AND parameters->>'max_tokens' != 'null'
					THEN (parameters->>'max_tokens')::integer 
					ELSE NULL 
				END,
				top_p = CASE 
					WHEN parameters->>'top_p' IS NOT NULL AND parameters->>'top_p' != 'null'
					THEN (parameters->>'top_p')::double precision 
					ELSE NULL 
				END,
				frequency_penalty = CASE 
					WHEN parameters->>'frequency_penalty' IS NOT NULL AND parameters->>'frequency_penalty' != 'null'
					THEN (parameters->>'frequency_penalty')::double precision 
					ELSE NULL 
				END,
				presence_penalty = CASE 
					WHEN parameters->>'presence_penalty' IS NOT NULL AND parameters->>'presence_penalty' != 'null'
					THEN (parameters->>'presence_penalty')::double precision 
					ELSE NULL 
				END;
		`)
		if err != nil {
			return err
		}
	}

	return nil
}
