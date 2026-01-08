package migrations

import "github.com/jmoiron/sqlx"

func init() {
	m.addMigration(&migration{
		version: "20260108073718",
		up:      mig_20260108073718_mcp_human_approval_up,
		down:    mig_20260108073718_mcp_human_approval_down,
	})
}

func mig_20260108073718_mcp_human_approval_up(tx *sqlx.Tx) error {
	// Add tools_requiring_human_approval column to agent_mcp_servers table
	_, err := tx.Exec(`
		ALTER TABLE agent_mcp_servers 
		ADD COLUMN IF NOT EXISTS tools_requiring_human_approval JSONB DEFAULT '[]'::jsonb;
	`)
	if err != nil {
		return err
	}
	return nil
}

func mig_20260108073718_mcp_human_approval_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`ALTER TABLE agent_mcp_servers DROP COLUMN IF EXISTS tools_requiring_human_approval;`)
	return err
}
