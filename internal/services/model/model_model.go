package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/google/uuid"
)

// ModelParameters represents model parameters as a flexible JSON map
type ModelParameters map[string]interface{}

// Scan implements the sql.Scanner interface for database/sql
func (m *ModelParameters) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]interface{})
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ModelParameters", value)
	}

	return json.Unmarshal(bytes, m)
}

// Value implements the driver.Valuer interface for database/sql
func (m ModelParameters) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaler
func (m ModelParameters) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(m))
}

// UnmarshalJSON implements json.Unmarshaler
func (m *ModelParameters) UnmarshalJSON(data []byte) error {
	var params map[string]interface{}
	if err := json.Unmarshal(data, &params); err != nil {
		return err
	}
	*m = ModelParameters(params)
	return nil
}

// Model represents a model configuration
type Model struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ProjectID    uuid.UUID       `json:"project_id" db:"project_id"`
	ProviderType string          `json:"provider_type" db:"provider_type"`
	Name         string          `json:"name" db:"name"`
	ModelID      string          `json:"model_id" db:"model_id"` // e.g., "gpt-4.1", "gpt-4o"
	Parameters   ModelParameters `json:"parameters" db:"parameters"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// ModelWithProvider includes provider information
type ModelWithProvider struct {
	Model
	ProviderType string `json:"provider_type" db:"provider_type"`
}

// CreateModelRequest represents the request to create a new model
type CreateModelRequest struct {
	ProviderType string          `json:"provider_type" validate:"required,oneof=OpenAI Anthropic Gemini xAI"`
	Name         string          `json:"name" validate:"required,min=1,max=255"`
	ModelID      string          `json:"model_id" validate:"required,min=1,max=255"`
	Parameters   ModelParameters `json:"parameters,omitempty"`
}

// UpdateModelRequest represents the request to update a model
type UpdateModelRequest struct {
	ProviderType *string          `json:"provider_type,omitempty" validate:"omitempty,oneof=OpenAI Anthropic Gemini xAI"`
	Name         *string          `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	ModelID      *string          `json:"model_id,omitempty" validate:"omitempty,min=1,max=255"`
	Parameters   *ModelParameters `json:"parameters,omitempty"`
}
