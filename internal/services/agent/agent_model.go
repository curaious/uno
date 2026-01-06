package agent

import (
	"database/sql/driver"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/google/uuid"
)

// ToolFilters represents a list of tool names to filter from an MCP server
type ToolFilters []string

// Scan implements the sql.Scanner interface for database/sql
func (t *ToolFilters) Scan(value interface{}) error {
	if value == nil {
		*t = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ToolFilters", value)
	}

	return json.Unmarshal(bytes, t)
}

// Value implements the driver.Valuer interface for database/sql
func (t ToolFilters) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// MarshalJSON implements json.Marshaler
func (t ToolFilters) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(t))
}

// UnmarshalJSON implements json.Unmarshaler
func (t *ToolFilters) UnmarshalJSON(data []byte) error {
	var filters []string
	if err := json.Unmarshal(data, &filters); err != nil {
		return err
	}
	*t = ToolFilters(filters)
	return nil
}

// Agent represents an agent configuration
type Agent struct {
	ID                           uuid.UUID  `json:"id" db:"id"`
	ProjectID                    uuid.UUID  `json:"project_id" db:"project_id"`
	Name                         string     `json:"name" db:"name"`
	ModelID                      uuid.UUID  `json:"model_id" db:"model_id"`
	PromptID                     uuid.UUID  `json:"prompt_id" db:"prompt_id"`
	PromptLabel                  *string    `json:"prompt_label,omitempty" db:"prompt_label"`
	SchemaID                     *uuid.UUID `json:"schema_id,omitempty" db:"schema_id"`
	EnableHistory                bool       `json:"enable_history" db:"enable_history"`
	SummarizerType               *string    `json:"summarizer_type,omitempty" db:"summarizer_type"` // "llm" or "sliding_window"
	LLMSummarizerTokenThreshold  *int       `json:"llm_summarizer_token_threshold,omitempty" db:"llm_summarizer_token_threshold"`
	LLMSummarizerKeepRecentCount *int       `json:"llm_summarizer_keep_recent_count,omitempty" db:"llm_summarizer_keep_recent_count"`
	LLMSummarizerPromptID        *uuid.UUID `json:"llm_summarizer_prompt_id,omitempty" db:"llm_summarizer_prompt_id"`
	LLMSummarizerPromptLabel     *string    `json:"llm_summarizer_prompt_label,omitempty" db:"llm_summarizer_prompt_label"`
	LLMSummarizerModelID         *uuid.UUID `json:"llm_summarizer_model_id,omitempty" db:"llm_summarizer_model_id"`
	SlidingWindowKeepCount       *int       `json:"sliding_window_keep_count,omitempty" db:"sliding_window_keep_count"`
	CreatedAt                    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                    time.Time  `json:"updated_at" db:"updated_at"`
}

// AgentMCPServer represents the relationship between an agent and an MCP server with tool filters
type AgentMCPServer struct {
	AgentID     uuid.UUID   `json:"agent_id" db:"agent_id"`
	MCPServerID uuid.UUID   `json:"mcp_server_id" db:"mcp_server_id"`
	ToolFilters ToolFilters `json:"tool_filters" db:"tool_filters"`
}

// AgentWithDetails includes related information
type AgentWithDetails struct {
	Agent
	ModelName         string                  `json:"model_name" db:"model_name"`
	ModelProviderType *string                 `json:"model_provider_type,omitempty" db:"model_provider_type"`
	ModelModelID      *string                 `json:"model_model_id,omitempty" db:"model_model_id"`
	ModelParameters   *utils.RawMessage       `json:"model_parameters,omitempty" db:"model_parameters"`
	PromptName        string                  `json:"prompt_name" db:"prompt_name"`
	SchemaName        *string                 `json:"schema_name,omitempty" db:"schema_name"`
	SchemaData        *utils.RawMessage       `json:"schema_data,omitempty" db:"schema_data"`
	MCPServers        []*AgentMCPServerDetail `json:"mcp_servers"`
	// Summarizer model and provider fields (nullable, only populated when summarizer_type is 'llm')
	SummarizerPromptName            *string           `json:"summarizer_prompt_name,omitempty" db:"summarizer_prompt_name"`
	SummarizerModelModelID          *string           `json:"summarizer_model_model_id,omitempty" db:"summarizer_model_model_id"`
	SummarizerModelParameters       *utils.RawMessage `json:"summarizer_model_parameters,omitempty" db:"summarizer_model_parameters"`
	SummarizerProviderID            *uuid.UUID        `json:"summarizer_provider_id,omitempty" db:"summarizer_provider_id"`
	SummarizerProviderType          *string           `json:"summarizer_provider_type,omitempty" db:"summarizer_provider_type"`
	SummarizerProviderAPIKey        *string           `json:"summarizer_provider_api_key,omitempty" db:"summarizer_provider_api_key"`
	SummarizerProviderBaseURL       *string           `json:"summarizer_provider_base_url,omitempty" db:"summarizer_provider_base_url"`
	SummarizerProviderCustomHeaders *utils.RawMessage `json:"summarizer_provider_custom_headers,omitempty" db:"summarizer_provider_custom_headers"`
}

// AgentMCPServerDetail includes MCP server information
type AgentMCPServerDetail struct {
	AgentMCPServer
	MCPServerName string `json:"mcp_server_name" db:"mcp_server_name"`
}

// CreateAgentRequest represents the request to create a new agent
type CreateAgentRequest struct {
	Name                         string              `json:"name" validate:"required,min=1,max=255"`
	ModelID                      uuid.UUID           `json:"model_id" validate:"required"`
	PromptID                     uuid.UUID           `json:"prompt_id" validate:"required"`
	PromptLabel                  *string             `json:"prompt_label,omitempty" validate:"omitempty,oneof=production latest"`
	SchemaID                     *uuid.UUID          `json:"schema_id,omitempty"`
	EnableHistory                bool                `json:"enable_history"`
	SummarizerType               *string             `json:"summarizer_type,omitempty" validate:"omitempty,oneof=llm sliding_window none"`
	LLMSummarizerTokenThreshold  *int                `json:"llm_summarizer_token_threshold,omitempty"`
	LLMSummarizerKeepRecentCount *int                `json:"llm_summarizer_keep_recent_count,omitempty"`
	LLMSummarizerPromptID        *uuid.UUID          `json:"llm_summarizer_prompt_id,omitempty"`
	LLMSummarizerPromptLabel     *string             `json:"llm_summarizer_prompt_label,omitempty" validate:"omitempty,oneof=production latest"`
	LLMSummarizerModelID         *uuid.UUID          `json:"llm_summarizer_model_id,omitempty"`
	SlidingWindowKeepCount       *int                `json:"sliding_window_keep_count,omitempty"`
	MCPServers                   []AgentMCPServerReq `json:"mcp_servers" validate:"omitempty"`
}

// AgentMCPServerReq represents MCP server configuration in a request
type AgentMCPServerReq struct {
	MCPServerID uuid.UUID   `json:"mcp_server_id" validate:"required"`
	ToolFilters ToolFilters `json:"tool_filters,omitempty"`
}

// UpdateAgentRequest represents the request to update an agent
type UpdateAgentRequest struct {
	Name                         *string              `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	ModelID                      *uuid.UUID           `json:"model_id,omitempty"`
	PromptID                     *uuid.UUID           `json:"prompt_id,omitempty"`
	PromptLabel                  *string              `json:"prompt_label,omitempty" validate:"omitempty,oneof=production latest"`
	SchemaID                     *uuid.UUID           `json:"schema_id,omitempty"`
	ClearSchemaID                bool                 `json:"clear_schema_id,omitempty"`
	EnableHistory                *bool                `json:"enable_history,omitempty"`
	SummarizerType               *string              `json:"summarizer_type,omitempty" validate:"omitempty,oneof=llm sliding_window"`
	LLMSummarizerTokenThreshold  *int                 `json:"llm_summarizer_token_threshold,omitempty"`
	LLMSummarizerKeepRecentCount *int                 `json:"llm_summarizer_keep_recent_count,omitempty"`
	LLMSummarizerPromptID        *uuid.UUID           `json:"llm_summarizer_prompt_id,omitempty"`
	LLMSummarizerPromptLabel     *string              `json:"llm_summarizer_prompt_label,omitempty" validate:"omitempty,oneof=production latest"`
	LLMSummarizerModelID         *uuid.UUID           `json:"llm_summarizer_model_id,omitempty"`
	SlidingWindowKeepCount       *int                 `json:"sliding_window_keep_count,omitempty"`
	MCPServers                   *[]AgentMCPServerReq `json:"mcp_servers,omitempty"`
}
