package project

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a workspace or collection an agent can belong to
type Project struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	DefaultKey *string   `json:"default_key,omitempty" db:"default_key"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CreateProjectRequest captures payload for creating a project
type CreateProjectRequest struct {
	Name       string  `json:"name" validate:"required,min=1,max=255"`
	DefaultKey *string `json:"default_key,omitempty"`
}

// UpdateProjectRequest captures payload for updating a project
type UpdateProjectRequest struct {
	Name       *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	DefaultKey *string `json:"default_key,omitempty"`
}
