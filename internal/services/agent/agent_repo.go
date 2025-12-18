package agent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AgentRepo handles database operations for agents
type AgentRepo struct {
	db *sqlx.DB
}

// NewAgentRepo creates a new agent repository
func NewAgentRepo(db *sqlx.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

// Create creates a new agent
func (r *AgentRepo) Create(ctx context.Context, projectID uuid.UUID, req *CreateAgentRequest) (*Agent, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the agent
	query := `
		INSERT INTO agents (project_id, name, model_id, prompt_id, prompt_label, schema_id, enable_history, 
			summarizer_type, llm_summarizer_token_threshold, llm_summarizer_keep_recent_count, 
			llm_summarizer_prompt_id, llm_summarizer_prompt_label, llm_summarizer_model_id, sliding_window_keep_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, project_id, name, model_id, prompt_id, prompt_label, schema_id, enable_history,
			summarizer_type, llm_summarizer_token_threshold, llm_summarizer_keep_recent_count,
			llm_summarizer_prompt_id, llm_summarizer_prompt_label, llm_summarizer_model_id, sliding_window_keep_count, created_at, updated_at
	`

	var agent Agent
	// Default to "latest" if prompt_label is not provided
	promptLabel := req.PromptLabel
	if promptLabel == nil {
		defaultLabel := "latest"
		promptLabel = &defaultLabel
	}
	err = tx.GetContext(ctx, &agent, query, projectID, req.Name, req.ModelID, req.PromptID, promptLabel, req.SchemaID,
		req.EnableHistory, req.SummarizerType, req.LLMSummarizerTokenThreshold,
		req.LLMSummarizerKeepRecentCount, req.LLMSummarizerPromptID, req.LLMSummarizerPromptLabel, req.LLMSummarizerModelID, req.SlidingWindowKeepCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Create agent_mcp_servers relationships
	if len(req.MCPServers) > 0 {
		for _, mcpServer := range req.MCPServers {
			if mcpServer.ToolFilters == nil {
				mcpServer.ToolFilters = ToolFilters{}
			}
			_, err = tx.Exec(`
				INSERT INTO agent_mcp_servers (agent_id, mcp_server_id, tool_filters)
				VALUES ($1, $2, $3)
			`, agent.ID, mcpServer.MCPServerID, mcpServer.ToolFilters)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent MCP server relationship: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &agent, nil
}

// GetByID retrieves an agent by ID with all related information
func (r *AgentRepo) GetByID(ctx context.Context, projectID, id uuid.UUID) (*AgentWithDetails, error) {
	// Get agent with model and prompt names, and summarizer model/provider details
	query := `
		SELECT 
			a.id, a.project_id, a.name, a.model_id, a.prompt_id, a.prompt_label, a.schema_id,
			a.enable_history, a.summarizer_type, a.llm_summarizer_token_threshold,
			a.llm_summarizer_keep_recent_count, a.llm_summarizer_prompt_id, a.llm_summarizer_prompt_label,
			a.llm_summarizer_model_id, a.sliding_window_keep_count,
			a.created_at, a.updated_at,
			m.name as model_name,
			m.provider_type as model_provider_type,
			m.model_id as model_model_id,
			m.parameters as model_parameters,
			p.name as prompt_name,
			s.name as schema_name,
			s.schema as schema_data,
			sp_prompt.name as summarizer_prompt_name,
			sm.model_id as summarizer_model_model_id,
			sm.parameters as summarizer_model_parameters,
			NULL::uuid as summarizer_provider_id,
			sm.provider_type as summarizer_provider_type,
			NULL::text as summarizer_provider_api_key,
			NULL::text as summarizer_provider_base_url,
			NULL::jsonb as summarizer_provider_custom_headers
		FROM agents a
		LEFT JOIN models m ON a.model_id = m.id
		LEFT JOIN prompts p ON a.prompt_id = p.id
		LEFT JOIN schemas s ON a.schema_id = s.id
		LEFT JOIN prompts sp_prompt ON a.llm_summarizer_prompt_id = sp_prompt.id
		LEFT JOIN models sm ON a.llm_summarizer_model_id = sm.id
		WHERE a.id = $1 AND a.project_id = $2
	`

	var agent AgentWithDetails
	err := r.db.GetContext(ctx, &agent, query, id, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Get MCP servers for this agent
	mcpServersQuery := `
		SELECT 
			ams.agent_id, ams.mcp_server_id, ams.tool_filters,
			ms.name as mcp_server_name
		FROM agent_mcp_servers ams
		JOIN mcp_servers ms ON ams.mcp_server_id = ms.id
		WHERE ams.agent_id = $1
		ORDER BY ms.name
	`

	var mcpServers []*AgentMCPServerDetail
	err = r.db.SelectContext(ctx, &mcpServers, mcpServersQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent MCP servers: %w", err)
	}

	agent.MCPServers = mcpServers
	return &agent, nil
}

// GetByName retrieves an agent by name
func (r *AgentRepo) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentWithDetails, error) {
	// Get agent with model and prompt names, and summarizer model/provider details
	query := `
		SELECT 
			a.id, a.project_id, a.name, a.model_id, a.prompt_id, a.prompt_label, a.schema_id,
			a.enable_history, a.summarizer_type, a.llm_summarizer_token_threshold,
			a.llm_summarizer_keep_recent_count, a.llm_summarizer_prompt_id, a.llm_summarizer_prompt_label,
			a.llm_summarizer_model_id, a.sliding_window_keep_count,
			a.created_at, a.updated_at,
			m.name as model_name,
			m.provider_type as model_provider_type,
			m.model_id as model_model_id,
			m.parameters as model_parameters,
			p.name as prompt_name,
			s.name as schema_name,
			s.schema as schema_data,
			sp_prompt.name as summarizer_prompt_name,
			sm.model_id as summarizer_model_model_id,
			sm.parameters as summarizer_model_parameters,
			NULL::uuid as summarizer_provider_id,
			sm.provider_type as summarizer_provider_type,
			NULL::text as summarizer_provider_api_key,
			NULL::text as summarizer_provider_base_url,
			NULL::jsonb as summarizer_provider_custom_headers
		FROM agents a
		LEFT JOIN models m ON a.model_id = m.id
		LEFT JOIN prompts p ON a.prompt_id = p.id
		LEFT JOIN schemas s ON a.schema_id = s.id
		LEFT JOIN prompts sp_prompt ON a.llm_summarizer_prompt_id = sp_prompt.id
		LEFT JOIN models sm ON a.llm_summarizer_model_id = sm.id
		WHERE a.name = $1 AND a.project_id = $2
	`

	var agent AgentWithDetails
	err := r.db.GetContext(ctx, &agent, query, name, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Get MCP servers for this agent
	mcpServersQuery := `
		SELECT 
			ams.agent_id, ams.mcp_server_id, ams.tool_filters,
			ms.name as mcp_server_name
		FROM agent_mcp_servers ams
		JOIN mcp_servers ms ON ams.mcp_server_id = ms.id
		WHERE ams.agent_id = $1
		ORDER BY ms.name
	`

	var mcpServers []*AgentMCPServerDetail
	err = r.db.SelectContext(ctx, &mcpServers, mcpServersQuery, agent.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent MCP servers: %w", err)
	}

	agent.MCPServers = mcpServers

	return &agent, nil
}

// List retrieves all agents with optional filtering
func (r *AgentRepo) List(ctx context.Context, projectID uuid.UUID) ([]*AgentWithDetails, error) {
	query := `
		SELECT 
			a.id, a.project_id, a.name, a.model_id, a.prompt_id, a.prompt_label, a.schema_id,
			a.enable_history, a.summarizer_type, a.llm_summarizer_token_threshold,
			a.llm_summarizer_keep_recent_count, a.llm_summarizer_prompt_id, a.llm_summarizer_prompt_label,
			a.llm_summarizer_model_id, a.sliding_window_keep_count,
			a.created_at, a.updated_at,
			m.name as model_name,
			m.provider_type as model_provider_type,
			m.model_id as model_model_id,
			m.parameters as model_parameters,
			p.name as prompt_name,
			s.name as schema_name,
			s.schema as schema_data,
			sp_prompt.name as summarizer_prompt_name,
			sm.model_id as summarizer_model_model_id,
			sm.parameters as summarizer_model_parameters,
			NULL::uuid as summarizer_provider_id,
			sm.provider_type as summarizer_provider_type,
			NULL::text as summarizer_provider_api_key,
			NULL::text as summarizer_provider_base_url,
			NULL::jsonb as summarizer_provider_custom_headers
		FROM agents a
		LEFT JOIN models m ON a.model_id = m.id
		LEFT JOIN prompts p ON a.prompt_id = p.id
		LEFT JOIN schemas s ON a.schema_id = s.id
		LEFT JOIN prompts sp_prompt ON a.llm_summarizer_prompt_id = sp_prompt.id
		LEFT JOIN models sm ON a.llm_summarizer_model_id = sm.id
		WHERE a.project_id = $1
		ORDER BY a.created_at DESC
	`

	var agents []*AgentWithDetails
	err := r.db.SelectContext(ctx, &agents, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	// Get MCP servers for each agent
	for _, agent := range agents {
		mcpServersQuery := `
			SELECT 
				ams.agent_id, ams.mcp_server_id, ams.tool_filters,
				ms.name as mcp_server_name
			FROM agent_mcp_servers ams
			JOIN mcp_servers ms ON ams.mcp_server_id = ms.id
			WHERE ams.agent_id = $1
			ORDER BY ms.name
		`

		var mcpServers []*AgentMCPServerDetail
		err = r.db.SelectContext(ctx, &mcpServers, mcpServersQuery, agent.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get agent MCP servers: %w", err)
		}

		agent.MCPServers = mcpServers
	}

	return agents, nil
}

// Update updates an agent
func (r *AgentRepo) Update(ctx context.Context, projectID, id uuid.UUID, req *UpdateAgentRequest) (*Agent, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build dynamic query for agent update
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.ModelID != nil {
		setParts = append(setParts, fmt.Sprintf("model_id = $%d", argIndex))
		args = append(args, *req.ModelID)
		argIndex++
	}

	if req.PromptID != nil {
		setParts = append(setParts, fmt.Sprintf("prompt_id = $%d", argIndex))
		args = append(args, *req.PromptID)
		argIndex++
	}

	if req.PromptLabel != nil {
		setParts = append(setParts, fmt.Sprintf("prompt_label = $%d", argIndex))
		args = append(args, *req.PromptLabel)
		argIndex++
	}

	if req.ClearSchemaID {
		setParts = append(setParts, fmt.Sprintf("schema_id = $%d", argIndex))
		args = append(args, nil)
		argIndex++
	} else if req.SchemaID != nil {
		setParts = append(setParts, fmt.Sprintf("schema_id = $%d", argIndex))
		args = append(args, *req.SchemaID)
		argIndex++
	}

	if req.EnableHistory != nil {
		setParts = append(setParts, fmt.Sprintf("enable_history = $%d", argIndex))
		args = append(args, *req.EnableHistory)
		argIndex++
	}

	if req.SummarizerType != nil {
		setParts = append(setParts, fmt.Sprintf("summarizer_type = $%d", argIndex))
		args = append(args, *req.SummarizerType)
		argIndex++
	}

	if req.LLMSummarizerTokenThreshold != nil {
		setParts = append(setParts, fmt.Sprintf("llm_summarizer_token_threshold = $%d", argIndex))
		args = append(args, *req.LLMSummarizerTokenThreshold)
		argIndex++
	}

	if req.LLMSummarizerKeepRecentCount != nil {
		setParts = append(setParts, fmt.Sprintf("llm_summarizer_keep_recent_count = $%d", argIndex))
		args = append(args, *req.LLMSummarizerKeepRecentCount)
		argIndex++
	}

	if req.LLMSummarizerPromptID != nil {
		setParts = append(setParts, fmt.Sprintf("llm_summarizer_prompt_id = $%d", argIndex))
		args = append(args, *req.LLMSummarizerPromptID)
		argIndex++
	}

	if req.LLMSummarizerPromptLabel != nil {
		setParts = append(setParts, fmt.Sprintf("llm_summarizer_prompt_label = $%d", argIndex))
		args = append(args, *req.LLMSummarizerPromptLabel)
		argIndex++
	}

	if req.LLMSummarizerModelID != nil {
		setParts = append(setParts, fmt.Sprintf("llm_summarizer_model_id = $%d", argIndex))
		args = append(args, *req.LLMSummarizerModelID)
		argIndex++
	}

	if req.SlidingWindowKeepCount != nil {
		setParts = append(setParts, fmt.Sprintf("sliding_window_keep_count = $%d", argIndex))
		args = append(args, *req.SlidingWindowKeepCount)
		argIndex++
	}

	if len(setParts) > 0 {
		setParts = append(setParts, "updated_at = NOW()")
		args = append(args, projectID)
		args = append(args, id)
		argIndex += 1

		setClause := ""
		for i, part := range setParts {
			if i > 0 {
				setClause += ", "
			}
			setClause += part
		}

		query := fmt.Sprintf(`
			UPDATE agents
			SET %s
			WHERE id = $%d AND project_id = $%d
			RETURNING id, project_id, name, model_id, prompt_id, prompt_label, schema_id, enable_history,
				summarizer_type, llm_summarizer_token_threshold, llm_summarizer_keep_recent_count,
				llm_summarizer_prompt_id, llm_summarizer_prompt_label, llm_summarizer_model_id, sliding_window_keep_count, created_at, updated_at
		`, setClause, argIndex, argIndex-1)

		var agent Agent
		err = tx.GetContext(ctx, &agent, query, args...)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("agent not found")
			}
			return nil, fmt.Errorf("failed to update agent: %w", err)
		}

		// Update MCP servers if provided
		if req.MCPServers != nil {
			// Delete existing relationships
			_, err = tx.Exec(`DELETE FROM agent_mcp_servers WHERE agent_id = $1`, id)
			if err != nil {
				return nil, fmt.Errorf("failed to delete agent MCP servers: %w", err)
			}

			// Create new relationships
			for _, mcpServer := range *req.MCPServers {
				if mcpServer.ToolFilters == nil {
					mcpServer.ToolFilters = ToolFilters{}
				}
				_, err = tx.Exec(`
					INSERT INTO agent_mcp_servers (agent_id, mcp_server_id, tool_filters)
					VALUES ($1, $2, $3)
				`, id, mcpServer.MCPServerID, mcpServer.ToolFilters)
				if err != nil {
					return nil, fmt.Errorf("failed to create agent MCP server relationship: %w", err)
				}
			}
		}

		if err = tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return &agent, nil
	}

	// If no fields to update and no MCP servers to update, just return the agent
	agent, err := r.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}

	// Convert AgentWithDetails to Agent for return type
	return &Agent{
		ID:        agent.ID,
		ProjectID: agent.ProjectID,
		Name:      agent.Name,
		ModelID:   agent.ModelID,
		PromptID:  agent.PromptID,
		CreatedAt: agent.CreatedAt,
		UpdatedAt: agent.UpdatedAt,
	}, nil
}

// Delete deletes an agent
func (r *AgentRepo) Delete(ctx context.Context, projectID, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete agent_mcp_servers relationships first (cascade should handle this, but being explicit)
	_, err = tx.Exec(`DELETE FROM agent_mcp_servers WHERE agent_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent MCP servers: %w", err)
	}

	// Delete the agent
	query := `DELETE FROM agents WHERE id = $1 AND project_id = $2`
	result, err := tx.ExecContext(ctx, query, id, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("agent not found")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
