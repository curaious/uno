package migrations

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	m.addMigration(&migration{
		version: "20260102120054",
		up:      mig_20260102120054_users_up,
		down:    mig_20260102120054_users_down,
	})
}

func mig_20260102120054_users_up(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            email VARCHAR(255) NOT NULL UNIQUE,
            password_hash TEXT,
            password_auth_enabled BOOLEAN DEFAULT TRUE,
            role VARCHAR(50) NOT NULL CHECK (role IN ('super-admin', 'project-admin', 'project-member', 'project-viewer')),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
    `)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
        CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    `)
	if err != nil {
		return err
	}

	// Seed with default super-admin
	password := "admin"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	_, err = tx.Exec(`
        INSERT INTO users (name, email, password_hash, password_auth_enabled, role)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (email) DO NOTHING;
    `, "Super Admin", "admin@admin.com", string(hashedPassword), true, "super-admin")

	return err
}

func mig_20260102120054_users_down(tx *sqlx.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS users;`)
	return err
}
