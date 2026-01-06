package provider

import (
	"database/sql/driver"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/curaious/uno/pkg/llm"
	"github.com/google/uuid"
)

// CustomHeadersMap represents a map of custom headers that can be stored in JSONB
type CustomHeadersMap map[string]string

// Scan implements the sql.Scanner interface for database/sql
func (h *CustomHeadersMap) Scan(value interface{}) error {
	if value == nil {
		*h = make(map[string]string)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into CustomHeadersMap", value)
	}

	return json.Unmarshal(bytes, h)
}

// Value implements the driver.Valuer interface for database/sql
func (h CustomHeadersMap) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}
	return json.Marshal(h)
}

// MarshalJSON implements json.Marshaler
func (h CustomHeadersMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(h))
}

// UnmarshalJSON implements json.Unmarshaler
func (h *CustomHeadersMap) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*h = CustomHeadersMap(m)
	return nil
}

// ProviderConfig represents provider-level configuration (base URL, custom headers)
type ProviderConfig struct {
	ProviderType  llm.ProviderName `json:"provider_type" db:"provider_type"`
	BaseURL       *string          `json:"base_url,omitempty" db:"base_url"`
	CustomHeaders CustomHeadersMap `json:"custom_headers,omitempty" db:"custom_headers"`
	CreatedAt     time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at" db:"updated_at"`
}

// APIKey represents an API key configuration
type APIKey struct {
	ID           uuid.UUID        `json:"id" db:"id"`
	ProviderType llm.ProviderName `json:"provider_type" db:"provider_type"`
	Name         string           `json:"name" db:"name"`
	APIKey       string           `json:"api_key" db:"api_key"`
	Enabled      bool             `json:"enabled" db:"enabled"`
	IsDefault    bool             `json:"is_default" db:"is_default"`
	CreatedAt    time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at" db:"updated_at"`
}

// CreateProviderConfigRequest represents the request to create/update provider config
type CreateProviderConfigRequest struct {
	ProviderType  llm.ProviderName `json:"provider_type" validate:"required,oneof=OpenAI Anthropic Gemini xAI"`
	BaseURL       *string          `json:"base_url,omitempty" validate:"omitempty,url"`
	CustomHeaders CustomHeadersMap `json:"custom_headers,omitempty"`
}

// UpdateProviderConfigRequest represents the request to update provider config
type UpdateProviderConfigRequest struct {
	BaseURL       *string           `json:"base_url,omitempty" validate:"omitempty,url"`
	CustomHeaders *CustomHeadersMap `json:"custom_headers,omitempty"`
}

// CreateAPIKeyRequest represents the request to create a new API key
type CreateAPIKeyRequest struct {
	ProviderType llm.ProviderName `json:"provider_type" validate:"required,oneof=OpenAI Anthropic Gemini xAI"`
	Name         string           `json:"name" validate:"required,min=1,max=255"`
	APIKey       string           `json:"api_key" validate:"required,min=1"`
	Enabled      bool             `json:"enabled,omitempty"`
	IsDefault    bool             `json:"is_default,omitempty"`
}

// UpdateAPIKeyRequest represents the request to update an API key
type UpdateAPIKeyRequest struct {
	Name      *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	APIKey    *string `json:"api_key,omitempty" validate:"omitempty,min=1"`
	Enabled   *bool   `json:"enabled,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
}
