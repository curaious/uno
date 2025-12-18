package prompt

import (
	"time"

	"github.com/google/uuid"
)

// Prompt represents a prompt template
type Prompt struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// PromptVersion represents a version of a prompt template
type PromptVersion struct {
	ID            uuid.UUID `json:"id" db:"id"`
	PromptID      uuid.UUID `json:"prompt_id" db:"prompt_id"`
	Version       int       `json:"version" db:"version"`
	Template      string    `json:"template" db:"template"`
	CommitMessage string    `json:"commit_message" db:"commit_message"`
	Label         *string   `json:"label,omitempty" db:"label"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// PromptWithLatestVersion combines a prompt with its latest version information
type PromptWithLatestVersion struct {
	Prompt
	LatestVersion   *int    `json:"latest_version,omitempty"`
	LatestCommitMsg *string `json:"latest_commit_message,omitempty"`
	LatestLabel     *string `json:"latest_label,omitempty"`
}

// CreatePromptRequest represents the request to create a new prompt
type CreatePromptRequest struct {
	Name          string  `json:"name" validate:"required,min=1,max=255"`
	Template      string  `json:"template" validate:"required,min=1"`
	CommitMessage string  `json:"commit_message" validate:"required,min=1,max=500"`
	Label         *string `json:"label,omitempty" validate:"omitempty,oneof=production latest"`
}

// CreatePromptVersionRequest represents the request to create a new prompt version
type CreatePromptVersionRequest struct {
	Template      string  `json:"template" validate:"required,min=1"`
	CommitMessage string  `json:"commit_message" validate:"required,min=1,max=500"`
	Label         *string `json:"label,omitempty" validate:"omitempty,oneof=production latest"`
}

// UpdatePromptVersionLabelRequest represents the request to update a prompt version label
type UpdatePromptVersionLabelRequest struct {
	Label *string `json:"label,omitempty" validate:"omitempty,oneof=production latest"`
}

// PromptVersionWithPrompt combines a prompt version with its prompt information
type PromptVersionWithPrompt struct {
	PromptVersion
	PromptName string `json:"prompt_name" db:"prompt_name"`
}
