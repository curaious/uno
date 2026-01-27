package agent_config

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/curaious/uno/internal/utils"
	"github.com/google/uuid"
)

// ModelConfig represents model configuration embedded in agent config
type ModelConfig struct {
	ProviderType string                 `json:"provider_type"`        // e.g., "OpenAI", "Anthropic", "Gemini", "xAI"
	ModelID      string                 `json:"model_id"`             // e.g., "gpt-4.1", "gpt-4o"
	Parameters   map[string]interface{} `json:"parameters,omitempty"` // Model parameters like temperature, max_tokens
}

// PromptConfig represents prompt configuration - either raw text or a reference to a prompt version
type PromptConfig struct {
	// Use either RawPrompt OR (PromptID + Label), not both
	RawPrompt *string    `json:"raw_prompt,omitempty"` // Raw prompt text
	PromptID  *uuid.UUID `json:"prompt_id,omitempty"`  // Reference to a prompt
	Version   *int       `json:"version,omitempty"`
}

// SchemaConfig represents output schema configuration
type SchemaConfig struct {
	Name          string            `json:"name"`
	Description   *string           `json:"description,omitempty"`
	Schema        *utils.RawMessage `json:"schema,omitempty"`      // JSON Schema definition
	SourceType    *string           `json:"source_type,omitempty"` // "manual", "go_struct", "typescript"
	SourceContent *string           `json:"source_content,omitempty"`
}

// MCPServerConfig represents an MCP server configuration with tool filters
type MCPServerConfig struct {
	Name                        string            `json:"name"`
	Endpoint                    string            `json:"endpoint"`
	Headers                     map[string]string `json:"headers,omitempty"`
	ToolFilters                 []string          `json:"tool_filters,omitempty"`                   // Tools to include (empty means all)
	ToolsRequiringHumanApproval []string          `json:"tools_requiring_human_approval,omitempty"` // Tools that need human approval
}

// SummarizerConfig represents conversation summarization configuration
type SummarizerConfig struct {
	Type                   string        `json:"type"`                                // "llm", "sliding_window", or "none"
	LLMTokenThreshold      *int          `json:"llm_token_threshold,omitempty"`       // For "llm" type
	LLMKeepRecentCount     *int          `json:"llm_keep_recent_count,omitempty"`     // For "llm" type
	LLMSummarizerPrompt    *PromptConfig `json:"llm_summarizer_prompt,omitempty"`     // For "llm" type
	LLMSummarizerModel     *ModelConfig  `json:"llm_summarizer_model,omitempty"`      // For "llm" type
	SlidingWindowKeepCount *int          `json:"sliding_window_keep_count,omitempty"` // For "sliding_window" type
}

// HistoryConfig represents conversation history configuration
type HistoryConfig struct {
	Enabled    bool              `json:"enabled"`
	Summarizer *SummarizerConfig `json:"summarizer,omitempty"` // Required when enabled is true
}

// ToolConfig represents tools enabled and their parameters
type ToolConfig struct {
	ImageGeneration *ImageGenerationToolConfig `json:"image_generation,omitempty"`
	WebSearch       *WebSearchToolConfig       `json:"web_search,omitempty"`
	CodeExecution   *CodeExecutionToolConfig   `json:"code_execution,omitempty"`
	Sandbox         *SandboxToolConfig         `json:"sandbox,omitempty"`
}

// ImageGenerationToolConfig represents parameters for the image generation tool
type ImageGenerationToolConfig struct {
	Enabled bool `json:"enabled"`
}

// WebSearchToolConfig represents parameters for the web search tool
type WebSearchToolConfig struct {
	Enabled bool `json:"enabled"`
}

// CodeExecutionToolConfig represents parameters for the code execution tool
type CodeExecutionToolConfig struct {
	Enabled bool `json:"enabled"`
}

// SandboxToolConfig represents parameters for the sandbox tool
type SandboxToolConfig struct {
	Enabled     bool    `json:"enabled"`
	DockerImage *string `json:"docker_image,omitempty"` // Optional Docker container image
}

// AgentConfigData represents the complete JSON configuration stored in the config column
type AgentConfigData struct {
	MaxIteration *int              `json:"max_iteration,omitempty"`
	Runtime      *string           `json:"runtime,omitempty"` // "Local", "Restate", or "Temporal"
	Model        *ModelConfig      `json:"model,omitempty"`
	Prompt       *PromptConfig     `json:"prompt,omitempty"`
	Schema       *SchemaConfig     `json:"schema,omitempty"`
	MCPServers   []MCPServerConfig `json:"mcp_servers,omitempty"`
	History      *HistoryConfig    `json:"history,omitempty"`
	Tools        *ToolConfig       `json:"tools,omitempty"`
}

// Scan implements the sql.Scanner interface for database/sql
func (c *AgentConfigData) Scan(value interface{}) error {
	if value == nil {
		*c = AgentConfigData{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into AgentConfigData", value)
	}

	return json.Unmarshal(bytes, c)
}

// Value implements the driver.Valuer interface for database/sql
func (c AgentConfigData) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// AgentConfig represents a versioned agent configuration stored in the database
type AgentConfig struct {
	ID        uuid.UUID       `json:"id" db:"id"`             // Row ID (unique per row)
	AgentID   uuid.UUID       `json:"agent_id" db:"agent_id"` // Stable UUID per agent (shared across versions)
	ProjectID uuid.UUID       `json:"project_id" db:"project_id"`
	Name      string          `json:"name" db:"name"`
	Version   int             `json:"version" db:"version"`     // Version number (0 is mutable, 1+ are immutable)
	Immutable bool            `json:"immutable" db:"immutable"` // true for versions > 0, false for version 0
	Config    AgentConfigData `json:"config" db:"config"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// CreateAgentConfigRequest represents the request to create a new agent config
type CreateAgentConfigRequest struct {
	Name   string          `json:"name" validate:"required,min=1,max=255"`
	Config AgentConfigData `json:"config" validate:"required"`
}

// UpdateAgentConfigRequest represents the request to update version 0 (mutable) or create a new version
type UpdateAgentConfigRequest struct {
	Config AgentConfigData `json:"config" validate:"required"`
}

// CreateVersionRequest represents the request to create a new immutable version from version 0
type CreateVersionRequest struct {
	// No fields needed - creates a new version from current version 0
}

// AgentConfigSummary represents a summary of an agent config for listing
type AgentConfigSummary struct {
	ID            uuid.UUID `json:"id" db:"id"`             // Row ID of version 0
	AgentID       uuid.UUID `json:"agent_id" db:"agent_id"` // Stable UUID per agent
	ProjectID     uuid.UUID `json:"project_id" db:"project_id"`
	Name          string    `json:"name" db:"name"`
	LatestVersion int       `json:"latest_version" db:"latest_version"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AgentConfigAlias represents a named mapping to one or two agent versions
type AgentConfigAlias struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`
	AgentID   uuid.UUID `json:"agent_id" db:"agent_id"`
	Name      string    `json:"name" db:"name"`
	Version1  int       `json:"version1" db:"version1"`           // Required: at least one version
	Version2  *int      `json:"version2,omitempty" db:"version2"` // Optional: second version
	Weight    *int      `json:"weight,omitempty" db:"weight"`     // Weight for version2 (0-100), required if version2 is set
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateAliasRequest represents the request to create a new alias
type CreateAliasRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Version1 int    `json:"version1" validate:"required"`
	Version2 *int   `json:"version2,omitempty"`
	Weight   *int   `json:"weight,omitempty" validate:"omitempty,min=0,max=100"`
}

// UpdateAliasRequest represents the request to update an existing alias
type UpdateAliasRequest struct {
	Name     string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Version1 *int   `json:"version1,omitempty"`
	Version2 *int   `json:"version2,omitempty"`
	Weight   *int   `json:"weight,omitempty" validate:"omitempty,min=0,max=100"`
}
