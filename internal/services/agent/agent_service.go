package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// AgentService handles business logic for agents
type AgentService struct {
	repo *AgentRepo
}

// NewAgentService creates a new agent service
func NewAgentService(repo *AgentRepo) *AgentService {
	return &AgentService{repo: repo}
}

// Create creates a new agent
func (s *AgentService) Create(ctx context.Context, projectID uuid.UUID, req *CreateAgentRequest) (*Agent, error) {
	// Check if agent with same name already exists in this project
	existing, err := s.repo.GetByName(ctx, projectID, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("agent with name '%s' already exists", req.Name)
	}

	// Validate that all MCP server IDs are unique
	if len(req.MCPServers) > 0 {
		mcpServerIDs := make(map[uuid.UUID]bool)
		for _, mcpServer := range req.MCPServers {
			if mcpServerIDs[mcpServer.MCPServerID] {
				return nil, fmt.Errorf("duplicate MCP server ID: %s", mcpServer.MCPServerID)
			}
			mcpServerIDs[mcpServer.MCPServerID] = true
		}
	}

	// Validate conversation history and summarizer configuration
	if err := s.validateHistoryConfig(req.EnableHistory, req.SummarizerType, req.LLMSummarizerTokenThreshold, req.LLMSummarizerKeepRecentCount, req.LLMSummarizerPromptID, req.LLMSummarizerModelID, req.SlidingWindowKeepCount); err != nil {
		return nil, err
	}

	// Create the agent
	agent, err := s.repo.Create(ctx, projectID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// GetByID retrieves an agent by ID
func (s *AgentService) GetByID(ctx context.Context, projectID, id uuid.UUID) (*AgentWithDetails, error) {
	agent, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// GetByName retrieves an agent by name
func (s *AgentService) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentWithDetails, error) {
	agent, err := s.repo.GetByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// List retrieves all agents
func (s *AgentService) List(ctx context.Context, projectID uuid.UUID) ([]*AgentWithDetails, error) {
	agents, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// Update updates an agent
func (s *AgentService) Update(ctx context.Context, projectID, id uuid.UUID, req *UpdateAgentRequest) (*Agent, error) {
	// Check if agent exists
	existing, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing agent: %w", err)
	}

	// If updating name, check if new name already exists
	if req.Name != nil && *req.Name != existing.Name {
		existingByName, err := s.repo.GetByName(ctx, projectID, *req.Name)
		if err == nil && existingByName != nil {
			return nil, fmt.Errorf("agent with name '%s' already exists", *req.Name)
		}
	}

	// Validate MCP servers if provided
	if req.MCPServers != nil {
		mcpServerIDs := make(map[uuid.UUID]bool)
		for _, mcpServer := range *req.MCPServers {
			if mcpServerIDs[mcpServer.MCPServerID] {
				return nil, fmt.Errorf("duplicate MCP server ID: %s", mcpServer.MCPServerID)
			}
			mcpServerIDs[mcpServer.MCPServerID] = true
		}
	}

	// Determine effective values for validation (use request values if provided, otherwise existing values)
	enableHistory := existing.EnableHistory
	if req.EnableHistory != nil {
		enableHistory = *req.EnableHistory
	}

	// If enable_history is being set to false, allow summarizer fields to be cleared
	// Otherwise, use request values if provided, or existing values
	var summarizerType *string
	var llmTokenThreshold *int
	var llmKeepRecentCount *int
	var llmPromptID *uuid.UUID
	var slidingWindowKeepCount *int

	if enableHistory {
		// History is enabled, so use request values if provided, otherwise existing values
		summarizerType = existing.SummarizerType
		if req.SummarizerType != nil {
			summarizerType = req.SummarizerType
		}

		llmTokenThreshold = existing.LLMSummarizerTokenThreshold
		if req.LLMSummarizerTokenThreshold != nil {
			llmTokenThreshold = req.LLMSummarizerTokenThreshold
		}

		llmKeepRecentCount = existing.LLMSummarizerKeepRecentCount
		if req.LLMSummarizerKeepRecentCount != nil {
			llmKeepRecentCount = req.LLMSummarizerKeepRecentCount
		}

		llmPromptID = existing.LLMSummarizerPromptID
		if req.LLMSummarizerPromptID != nil {
			llmPromptID = req.LLMSummarizerPromptID
		}

		slidingWindowKeepCount = existing.SlidingWindowKeepCount
		if req.SlidingWindowKeepCount != nil {
			slidingWindowKeepCount = req.SlidingWindowKeepCount
		}
	} else {
		// History is disabled, use request values if provided (allows clearing), otherwise nil
		if req.SummarizerType != nil {
			summarizerType = req.SummarizerType
		}
		if req.LLMSummarizerTokenThreshold != nil {
			llmTokenThreshold = req.LLMSummarizerTokenThreshold
		}
		if req.LLMSummarizerKeepRecentCount != nil {
			llmKeepRecentCount = req.LLMSummarizerKeepRecentCount
		}
		if req.LLMSummarizerPromptID != nil {
			llmPromptID = req.LLMSummarizerPromptID
		}
		if req.SlidingWindowKeepCount != nil {
			slidingWindowKeepCount = req.SlidingWindowKeepCount
		}
	}

	// Get llmModelID for validation
	var llmModelID *uuid.UUID
	if enableHistory {
		if req.LLMSummarizerModelID != nil {
			llmModelID = req.LLMSummarizerModelID
		} else if existing.LLMSummarizerModelID != nil {
			llmModelID = existing.LLMSummarizerModelID
		}
	}

	// Validate conversation history and summarizer configuration
	if err := s.validateHistoryConfig(enableHistory, summarizerType, llmTokenThreshold, llmKeepRecentCount, llmPromptID, llmModelID, slidingWindowKeepCount); err != nil {
		return nil, err
	}

	// Update the agent
	agent, err := s.repo.Update(ctx, projectID, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return agent, nil
}

// Delete deletes an agent
func (s *AgentService) Delete(ctx context.Context, projectID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, projectID, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// validateHistoryConfig validates conversation history and summarizer configuration
func (s *AgentService) validateHistoryConfig(
	enableHistory bool,
	summarizerType *string,
	llmTokenThreshold *int,
	llmKeepRecentCount *int,
	llmPromptID *uuid.UUID,
	llmModelID *uuid.UUID,
	slidingWindowKeepCount *int,
) error {
	// If history is enabled, summarizer type must be provided
	if enableHistory {
		if summarizerType == nil || *summarizerType == "" {
			return fmt.Errorf("summarizer_type is required when enable_history is true")
		}

		// Validate that summarizer type is one of the allowed values
		if *summarizerType != "llm" && *summarizerType != "sliding_window" && *summarizerType != "none" {
			return fmt.Errorf("summarizer_type must be either 'llm', 'sliding_window', or 'none'")
		}

		// Validate LLM summarizer configuration
		if *summarizerType == "llm" {
			if llmTokenThreshold == nil || *llmTokenThreshold <= 0 {
				return fmt.Errorf("llm_summarizer_token_threshold is required and must be greater than 0 when summarizer_type is 'llm'")
			}
			if llmKeepRecentCount == nil || *llmKeepRecentCount < 0 {
				return fmt.Errorf("llm_summarizer_keep_recent_count is required and must be non-negative when summarizer_type is 'llm'")
			}
			if llmPromptID == nil {
				return fmt.Errorf("llm_summarizer_prompt_id is required when summarizer_type is 'llm'")
			}
			if llmModelID == nil {
				return fmt.Errorf("llm_summarizer_model_id is required when summarizer_type is 'llm'")
			}
		}

		// Validate sliding window summarizer configuration
		if *summarizerType == "sliding_window" {
			if slidingWindowKeepCount == nil || *slidingWindowKeepCount <= 0 {
				return fmt.Errorf("sliding_window_keep_count is required and must be greater than 0 when summarizer_type is 'sliding_window'")
			}
		}

		// "none" type means no summarization, so no additional validation needed
	}

	// Note: When enable_history is false, we allow summarizer fields to be nil/empty
	// This allows clearing the configuration when disabling history

	return nil
}
