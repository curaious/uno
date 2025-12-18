package schema

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SourceType represents the source from which a schema was generated
type SourceType string

const (
	SourceTypeManual     SourceType = "manual"     // Manually created via UI
	SourceTypeGoStruct   SourceType = "go_struct"  // Generated from Go struct (future)
	SourceTypeTypeScript SourceType = "typescript" // Generated from TypeScript interface (future)
)

// Schema represents a JSON schema with metadata
type Schema struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	ProjectID     uuid.UUID       `json:"project_id" db:"project_id"`
	Name          string          `json:"name" db:"name"`
	Description   *string         `json:"description,omitempty" db:"description"`
	Schema        json.RawMessage `json:"schema" db:"schema"`
	SourceType    SourceType      `json:"source_type" db:"source_type"`
	SourceContent *string         `json:"source_content,omitempty" db:"source_content"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// CreateSchemaRequest represents the request to create a new schema
type CreateSchemaRequest struct {
	Name          string          `json:"name" validate:"required,min=1,max=255"`
	Description   *string         `json:"description,omitempty"`
	Schema        json.RawMessage `json:"schema" validate:"required"`
	SourceType    SourceType      `json:"source_type,omitempty"`
	SourceContent *string         `json:"source_content,omitempty"`
}

// UpdateSchemaRequest represents the request to update a schema
type UpdateSchemaRequest struct {
	Name          *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description   *string         `json:"description,omitempty"`
	Schema        json.RawMessage `json:"schema,omitempty"`
	SourceType    *SourceType     `json:"source_type,omitempty"`
	SourceContent *string         `json:"source_content,omitempty"`
}

// JSONSchemaProperty represents a property in a JSON schema
type JSONSchemaProperty struct {
	Type        string                        `json:"type,omitempty"`
	Description string                        `json:"description,omitempty"`
	Properties  map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Items       *JSONSchemaProperty           `json:"items,omitempty"`
	Required    []string                      `json:"required,omitempty"`
	Enum        []interface{}                 `json:"enum,omitempty"`
	Default     interface{}                   `json:"default,omitempty"`
	Format      string                        `json:"format,omitempty"`
	Minimum     *float64                      `json:"minimum,omitempty"`
	Maximum     *float64                      `json:"maximum,omitempty"`
	MinLength   *int                          `json:"minLength,omitempty"`
	MaxLength   *int                          `json:"maxLength,omitempty"`
	Pattern     string                        `json:"pattern,omitempty"`
	Ref         string                        `json:"$ref,omitempty"`
}

// JSONSchema represents a full JSON schema document
type JSONSchema struct {
	Schema      string                        `json:"$schema,omitempty"`
	Type        string                        `json:"type"`
	Title       string                        `json:"title,omitempty"`
	Description string                        `json:"description,omitempty"`
	Properties  map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Required    []string                      `json:"required,omitempty"`
	Definitions map[string]JSONSchemaProperty `json:"definitions,omitempty"`
}
