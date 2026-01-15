package agent_config

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// AgentConfigService handles business logic for agent configs
type AgentConfigService struct {
	repo *AgentConfigRepo
}

// NewAgentConfigService creates a new agent config service
func NewAgentConfigService(repo *AgentConfigRepo) *AgentConfigService {
	return &AgentConfigService{repo: repo}
}

// Create creates a new agent config
func (s *AgentConfigService) Create(ctx context.Context, projectID uuid.UUID, req *CreateAgentConfigRequest) (*AgentConfig, error) {
	// Check if agent config with same name already exists
	exists, err := s.repo.Exists(ctx, projectID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing agent config: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("agent config with name '%s' already exists", req.Name)
	}

	// Validate the configuration
	if err := s.validateConfig(&req.Config); err != nil {
		return nil, err
	}

	// Create the agent config
	config, err := s.repo.Create(ctx, projectID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config: %w", err)
	}

	return config, nil
}

// CreateVersion creates a new version of an existing agent config
func (s *AgentConfigService) CreateVersion(ctx context.Context, projectID uuid.UUID, name string, req *UpdateAgentConfigRequest) (*AgentConfig, error) {
	// Check if agent config exists
	exists, err := s.repo.Exists(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing agent config: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("agent config with name '%s' not found", name)
	}

	// Validate the configuration
	if err := s.validateConfig(&req.Config); err != nil {
		return nil, err
	}

	// Create new version
	config, err := s.repo.CreateVersion(ctx, projectID, name, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config version: %w", err)
	}

	return config, nil
}

// GetByID retrieves an agent config by ID
func (s *AgentConfigService) GetByID(ctx context.Context, projectID, id uuid.UUID) (*AgentConfig, error) {
	config, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// GetByNameAndVersion retrieves an agent config by name and version
func (s *AgentConfigService) GetByNameAndVersion(ctx context.Context, projectID uuid.UUID, name string, version int) (*AgentConfig, error) {
	config, err := s.repo.GetByNameAndVersion(ctx, projectID, name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// GetLatestByName retrieves the latest version of an agent config by name
func (s *AgentConfigService) GetLatestByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentConfig, error) {
	config, err := s.repo.GetLatestByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// List retrieves all agent configs for a project
func (s *AgentConfigService) List(ctx context.Context, projectID uuid.UUID) ([]*AgentConfigSummary, error) {
	configs, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent configs: %w", err)
	}

	return configs, nil
}

// ListVersions retrieves all versions of an agent config by name
func (s *AgentConfigService) ListVersions(ctx context.Context, projectID uuid.UUID, name string) ([]*AgentConfig, error) {
	configs, err := s.repo.ListVersions(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent config versions: %w", err)
	}

	return configs, nil
}

// Delete deletes all versions of an agent config by name
func (s *AgentConfigService) Delete(ctx context.Context, projectID uuid.UUID, name string) error {
	err := s.repo.Delete(ctx, projectID, name)
	if err != nil {
		return fmt.Errorf("failed to delete agent config: %w", err)
	}

	return nil
}

// DeleteVersion deletes a specific version of an agent config
func (s *AgentConfigService) DeleteVersion(ctx context.Context, projectID uuid.UUID, name string, version int) error {
	err := s.repo.DeleteVersion(ctx, projectID, name, version)
	if err != nil {
		return fmt.Errorf("failed to delete agent config version: %w", err)
	}

	return nil
}

// validateConfig validates the agent configuration
func (s *AgentConfigService) validateConfig(config *AgentConfigData) error {
	// Validate model config if provided
	if config.Model != nil {
		if config.Model.ProviderType == "" {
			return fmt.Errorf("model provider_type is required")
		}
		if config.Model.ModelID == "" {
			return fmt.Errorf("model model_id is required")
		}
	}

	// Validate prompt config if provided
	if config.Prompt != nil {
		hasRawPrompt := config.Prompt.RawPrompt != nil && *config.Prompt.RawPrompt != ""
		hasPromptID := config.Prompt.PromptID != nil

		if hasRawPrompt && hasPromptID {
			return fmt.Errorf("prompt cannot have both raw_prompt and prompt_id")
		}
		if !hasRawPrompt && !hasPromptID {
			return fmt.Errorf("prompt must have either raw_prompt or prompt_id")
		}
	}

	// Validate MCP server configs if provided
	for i, mcpServer := range config.MCPServers {
		if mcpServer.Name == "" {
			return fmt.Errorf("mcp_servers[%d].name is required", i)
		}
		if mcpServer.Endpoint == "" {
			return fmt.Errorf("mcp_servers[%d].endpoint is required", i)
		}
	}

	// Validate history config if provided
	if config.History != nil && config.History.Enabled {
		if config.History.Summarizer == nil {
			return fmt.Errorf("history.summarizer is required when history is enabled")
		}

		summarizer := config.History.Summarizer
		if summarizer.Type == "" {
			return fmt.Errorf("history.summarizer.type is required")
		}

		switch summarizer.Type {
		case "llm":
			if summarizer.LLMTokenThreshold == nil || *summarizer.LLMTokenThreshold <= 0 {
				return fmt.Errorf("history.summarizer.llm_token_threshold is required and must be > 0 for llm type")
			}
			if summarizer.LLMKeepRecentCount == nil || *summarizer.LLMKeepRecentCount < 0 {
				return fmt.Errorf("history.summarizer.llm_keep_recent_count is required and must be >= 0 for llm type")
			}
			if summarizer.LLMSummarizerPrompt == nil {
				return fmt.Errorf("history.summarizer.llm_summarizer_prompt is required for llm type")
			}
			if summarizer.LLMSummarizerModel == nil {
				return fmt.Errorf("history.summarizer.llm_summarizer_model is required for llm type")
			}
		case "sliding_window":
			if summarizer.SlidingWindowKeepCount == nil || *summarizer.SlidingWindowKeepCount <= 0 {
				return fmt.Errorf("history.summarizer.sliding_window_keep_count is required and must be > 0 for sliding_window type")
			}
		case "none":
			// No additional validation needed
		default:
			return fmt.Errorf("history.summarizer.type must be 'llm', 'sliding_window', or 'none'")
		}
	}

	return nil
}
