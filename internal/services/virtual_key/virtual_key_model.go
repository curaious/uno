package virtual_key

import (
	"database/sql/driver"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/praveen001/uno/pkg/llm"
)

// RateLimit represents a single rate limit configuration
type RateLimit struct {
	Unit  string `json:"unit" validate:"required,oneof=1min 1h 6h 12h 1d 1w 1mo"`
	Limit int64  `json:"limit" validate:"required,min=1"`
}

// RateLimits represents a list of rate limits that can be stored in JSONB
type RateLimits []RateLimit

// Scan implements the sql.Scanner interface for database/sql
func (r *RateLimits) Scan(value interface{}) error {
	if value == nil {
		*r = []RateLimit{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into RateLimits", value)
	}

	return json.Unmarshal(bytes, r)
}

// Value implements the driver.Valuer interface for database/sql
func (r RateLimits) Value() (driver.Value, error) {
	if r == nil {
		return nil, nil
	}
	return json.Marshal(r)
}

// MarshalJSON implements json.Marshaler
func (r RateLimits) MarshalJSON() ([]byte, error) {
	return json.Marshal([]RateLimit(r))
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RateLimits) UnmarshalJSON(data []byte) error {
	var limits []RateLimit
	if err := json.Unmarshal(data, &limits); err != nil {
		return err
	}
	*r = RateLimits(limits)
	return nil
}

// VirtualKey represents a virtual key configuration
type VirtualKey struct {
	ID         uuid.UUID          `json:"id" db:"id"`
	Name       string             `json:"name" db:"name"`
	SecretKey  string             `json:"secret_key" db:"secret_key"`
	Providers  []llm.ProviderName `json:"providers" db:"-"`
	ModelNames []string           `json:"model_ids" db:"-"` // Keep json tag as model_ids for API compatibility
	RateLimits RateLimits         `json:"rate_limits" db:"rate_limits"`
	CreatedAt  time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" db:"updated_at"`
}

// CreateVirtualKeyRequest represents the request to create a new virtual key
type CreateVirtualKeyRequest struct {
	Name       string             `json:"name" validate:"required,min=1,max=255"`
	Providers  []llm.ProviderName `json:"providers" validate:"dive,oneof=OpenAI Anthropic Gemini xAI"`
	ModelIDs   []string           `json:"model_ids,omitempty"` // Changed to []string for model names
	RateLimits *RateLimits        `json:"rate_limits,omitempty"`
}

// UpdateVirtualKeyRequest represents the request to update a virtual key
type UpdateVirtualKeyRequest struct {
	Name       *string             `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Providers  *[]llm.ProviderName `json:"providers,omitempty" validate:"omitempty,dive,oneof=OpenAI Anthropic Gemini xAI"`
	ModelIDs   *[]string           `json:"model_ids,omitempty"` // Changed to *[]string for model names
	RateLimits *RateLimits         `json:"rate_limits,omitempty"`
}
