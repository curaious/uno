package virtual_key

import (
	"time"

	"github.com/google/uuid"
	"github.com/praveen001/uno/pkg/llm"
)

// VirtualKey represents a virtual key configuration
type VirtualKey struct {
	ID         uuid.UUID          `json:"id" db:"id"`
	Name       string             `json:"name" db:"name"`
	SecretKey  string             `json:"secret_key" db:"secret_key"`
	Providers  []llm.ProviderName `json:"providers" db:"-"`
	ModelNames []string           `json:"model_ids" db:"-"` // Keep json tag as model_ids for API compatibility
	CreatedAt  time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" db:"updated_at"`
}

// CreateVirtualKeyRequest represents the request to create a new virtual key
type CreateVirtualKeyRequest struct {
	Name      string             `json:"name" validate:"required,min=1,max=255"`
	Providers []llm.ProviderName `json:"providers" validate:"dive,oneof=OpenAI Anthropic Gemini xAI"`
	ModelIDs  []string           `json:"model_ids,omitempty"` // Changed to []string for model names
}

// UpdateVirtualKeyRequest represents the request to update a virtual key
type UpdateVirtualKeyRequest struct {
	Name      *string             `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Providers *[]llm.ProviderName `json:"providers,omitempty" validate:"omitempty,dive,oneof=OpenAI Anthropic Gemini xAI"`
	ModelIDs  *[]string           `json:"model_ids,omitempty"` // Changed to *[]string for model names
}
