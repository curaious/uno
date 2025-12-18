package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrProjectNotFound = errors.New("project not found")

// ProjectRepo handles database operations for projects
type ProjectRepo struct {
	db *sqlx.DB
}

// NewProjectRepo creates a new project repository
func NewProjectRepo(db *sqlx.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// Create creates a new project
func (r *ProjectRepo) Create(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	query := `
        INSERT INTO projects (name, default_key)
        VALUES ($1, $2)
        RETURNING id, name, default_key, created_at, updated_at
    `

	var project Project
	err := r.db.GetContext(ctx, &project, query, req.Name, req.DefaultKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return &project, nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	query := `
        SELECT id, name, default_key, created_at, updated_at
        FROM projects
        WHERE id = $1
    `

	var project Project
	err := r.db.GetContext(ctx, &project, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// GetByName retrieves a project by name
func (r *ProjectRepo) GetByName(ctx context.Context, name string) (*Project, error) {
	query := `
        SELECT id, name, default_key, created_at, updated_at
        FROM projects
        WHERE name = $1
    `

	var project Project
	err := r.db.GetContext(ctx, &project, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// List retrieves all projects ordered by creation date
func (r *ProjectRepo) List(ctx context.Context) ([]*Project, error) {
	query := `
        SELECT id, name, default_key, created_at, updated_at
        FROM projects
        ORDER BY created_at DESC
    `

	var projects []*Project
	err := r.db.SelectContext(ctx, &projects, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// Update updates project fields
func (r *ProjectRepo) Update(ctx context.Context, id uuid.UUID, req *UpdateProjectRequest) (*Project, error) {
	setParts := []string{}
	args := []interface{}{}

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", len(args)+1))
		args = append(args, *req.Name)
	}

	if req.DefaultKey != nil {
		setParts = append(setParts, fmt.Sprintf("default_key = $%d", len(args)+1))
		args = append(args, *req.DefaultKey)
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`
        UPDATE projects
        SET %s
        WHERE id = $%d
        RETURNING id, name, default_key, created_at, updated_at
    `, strings.Join(setParts, ", "), len(args))

	var project Project
	err := r.db.GetContext(ctx, &project, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return &project, nil
}

// Delete removes a project by ID
func (r *ProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProjectNotFound
	}

	return nil
}
