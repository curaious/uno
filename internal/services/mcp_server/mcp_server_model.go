package mcp_server

import (
	"database/sql/driver"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/google/uuid"
)

// HeadersMap represents a map of headers that can be stored in JSONB
type HeadersMap map[string]string

// Scan implements the sql.Scanner interface for database/sql
func (h *HeadersMap) Scan(value interface{}) error {
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
		return fmt.Errorf("cannot scan %T into HeadersMap", value)
	}

	return json.Unmarshal(bytes, h)
}

// Value implements the driver.Valuer interface for database/sql
func (h HeadersMap) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}
	return json.Marshal(h)
}

// MarshalJSON implements json.Marshaler
func (h HeadersMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(h))
}

// UnmarshalJSON implements json.Unmarshaler
func (h *HeadersMap) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*h = HeadersMap(m)
	return nil
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Endpoint  string     `json:"endpoint" db:"endpoint"`
	Headers   HeadersMap `json:"headers" db:"headers"`
	ProjectID uuid.UUID  `json:"project_id" db:"project_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateMCPServerRequest represents the request to create a new MCP server
type CreateMCPServerRequest struct {
	Name     string     `json:"name" validate:"required,min=1,max=255"`
	Endpoint string     `json:"endpoint" validate:"required,min=1,max=500,url"`
	Headers  HeadersMap `json:"headers" validate:"omitempty"`
}

// UpdateMCPServerRequest represents the request to update an MCP server
type UpdateMCPServerRequest struct {
	Name     *string     `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Endpoint *string     `json:"endpoint,omitempty" validate:"omitempty,min=1,max=500,url"`
	Headers  *HeadersMap `json:"headers,omitempty"`
}
