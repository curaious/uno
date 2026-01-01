package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, password_auth_enabled, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, password_auth_enabled, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}
