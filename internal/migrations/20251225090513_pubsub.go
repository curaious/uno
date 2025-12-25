package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20251225090513",
		up:      mig_20251225090513_pubsub_up,
		down:    mig_20251225090513_pubsub_down,
	})
}

func mig_20251225090513_pubsub_up(tx *sqlx.Tx) error {
	// Create a generic notify function that sends the table name and operation
	_, err := tx.Exec(`
		CREATE OR REPLACE FUNCTION notify_config_change()
		RETURNS TRIGGER AS $$
		DECLARE
			payload TEXT;
		BEGIN
			-- Build payload with table name and operation
			payload := TG_TABLE_NAME || ':' || TG_OP;
			PERFORM pg_notify('config_changes', payload);
			RETURN COALESCE(NEW, OLD);
		END;
		$$ LANGUAGE plpgsql;
	`)
	if err != nil {
		return err
	}

	// Trigger for provider_configs table
	_, err = tx.Exec(`
		CREATE TRIGGER provider_configs_notify
		AFTER INSERT OR UPDATE OR DELETE ON provider_configs
		FOR EACH ROW EXECUTE FUNCTION notify_config_change();
	`)
	if err != nil {
		return err
	}

	// Trigger for api_keys table
	_, err = tx.Exec(`
		CREATE TRIGGER api_keys_notify
		AFTER INSERT OR UPDATE OR DELETE ON api_keys
		FOR EACH ROW EXECUTE FUNCTION notify_config_change();
	`)
	if err != nil {
		return err
	}

	// Trigger for virtual_keys table
	_, err = tx.Exec(`
		CREATE TRIGGER virtual_keys_notify
		AFTER INSERT OR UPDATE OR DELETE ON virtual_keys
		FOR EACH ROW EXECUTE FUNCTION notify_config_change();
	`)
	if err != nil {
		return err
	}

	// Trigger for virtual_key_providers table
	_, err = tx.Exec(`
		CREATE TRIGGER virtual_key_providers_notify
		AFTER INSERT OR UPDATE OR DELETE ON virtual_key_providers
		FOR EACH ROW EXECUTE FUNCTION notify_config_change();
	`)
	if err != nil {
		return err
	}

	// Trigger for virtual_key_models table
	_, err = tx.Exec(`
		CREATE TRIGGER virtual_key_models_notify
		AFTER INSERT OR UPDATE OR DELETE ON virtual_key_models
		FOR EACH ROW EXECUTE FUNCTION notify_config_change();
	`)
	if err != nil {
		return err
	}

	return nil
}

func mig_20251225090513_pubsub_down(tx *sqlx.Tx) error {
	// Drop triggers
	_, err := tx.Exec(`DROP TRIGGER IF EXISTS provider_configs_notify ON provider_configs;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TRIGGER IF EXISTS api_keys_notify ON api_keys;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TRIGGER IF EXISTS virtual_keys_notify ON virtual_keys;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TRIGGER IF EXISTS virtual_key_providers_notify ON virtual_key_providers;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TRIGGER IF EXISTS virtual_key_models_notify ON virtual_key_models;`)
	if err != nil {
		return err
	}

	// Drop the function
	_, err = tx.Exec(`DROP FUNCTION IF EXISTS notify_config_change();`)
	if err != nil {
		return err
	}

	return nil
}
