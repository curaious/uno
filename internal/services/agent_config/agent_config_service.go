package agent_config

import (
	"context"
	"fmt"

	"github.com/curaious/uno/internal/utils"
	"github.com/google/uuid"
)

type FileSystem interface {
	CreateAgentDataDirectory(config *AgentConfig) error
}

// AgentConfigService handles business logic for agent configs
type AgentConfigService struct {
	repo *AgentConfigRepo
	fs   FileSystem
}

// NewAgentConfigService creates a new agent config service
func NewAgentConfigService(repo *AgentConfigRepo, fs FileSystem) *AgentConfigService {
	return &AgentConfigService{repo: repo, fs: fs}
}

// Create creates a new agent config with version 0
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

	if err = s.fs.CreateAgentDataDirectory(config); err != nil {
		return nil, fmt.Errorf("failed to create agent data directory: %w", err)
	}

	return config, nil
}

// UpdateVersion0 updates version 0 in place (mutable)
func (s *AgentConfigService) UpdateVersion0(ctx context.Context, agentID uuid.UUID, req *UpdateAgentConfigRequest) (*AgentConfig, error) {
	// Validate the configuration
	if err := s.validateConfig(&req.Config); err != nil {
		return nil, err
	}

	// Update version 0
	config, err := s.repo.UpdateVersion0(ctx, agentID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent config: %w", err)
	}

	if err = s.fs.CreateAgentDataDirectory(config); err != nil {
		return nil, fmt.Errorf("failed to create agent data directory: %w", err)
	}

	return config, nil
}

// UpdateVersion0ByName updates version 0 by name (for backward compatibility)
func (s *AgentConfigService) UpdateVersion0ByName(ctx context.Context, projectID uuid.UUID, name string, req *UpdateAgentConfigRequest) (*AgentConfig, error) {
	// Get agent_id
	agentID, err := s.repo.GetAgentIDByName(ctx, projectID, name)
	if err != nil {
		return nil, err
	}

	return s.UpdateVersion0(ctx, agentID, req)
}

// CreateVersion creates a new immutable version from version 0
func (s *AgentConfigService) CreateVersion(ctx context.Context, agentID uuid.UUID) (*AgentConfig, error) {
	// Create new immutable version
	config, err := s.repo.CreateVersion(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config version: %w", err)
	}

	if err = s.fs.CreateAgentDataDirectory(config); err != nil {
		return nil, fmt.Errorf("failed to create agent data directory: %w", err)
	}

	return config, nil
}

// CreateVersionByName creates a new immutable version by name (for backward compatibility)
func (s *AgentConfigService) CreateVersionByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentConfig, error) {
	// Get agent_id
	agentID, err := s.repo.GetAgentIDByName(ctx, projectID, name)
	if err != nil {
		return nil, err
	}

	return s.CreateVersion(ctx, agentID)
}

// GetByID retrieves an agent config by row ID
func (s *AgentConfigService) GetByID(ctx context.Context, projectID, id uuid.UUID) (*AgentConfig, error) {
	config, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// GetByAgentIDAndVersion retrieves an agent config by agent_id and version
func (s *AgentConfigService) GetByAgentIDAndVersion(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID, version int) (*AgentConfig, error) {
	config, err := s.repo.GetByAgentIDAndVersion(ctx, projectID, agentID, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// GetByNameAndVersion retrieves an agent config by name and version (for backward compatibility)
func (s *AgentConfigService) GetByNameAndVersion(ctx context.Context, projectID uuid.UUID, name string, version int) (*AgentConfig, error) {
	config, err := s.repo.GetByNameAndVersion(ctx, projectID, name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// GetLatestByName retrieves version 0 (the mutable version) of an agent config by name
func (s *AgentConfigService) GetLatestByName(ctx context.Context, projectID uuid.UUID, name string) (*AgentConfig, error) {
	config, err := s.repo.GetLatestByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent config: %w", err)
	}

	return config, nil
}

// GetAgentVersionByAlias retrieves an alias by agent name and alias name
func (s *AgentConfigService) GetAgentVersionByAlias(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID, aliasName string) (int, error) {
	alias, err := s.repo.GetAliasByName(ctx, projectID, agentID, aliasName)
	if err != nil {
		return -1, err
	}

	if alias.Version2 == nil {
		return alias.Version1, nil
	}

	idx := utils.WeightedRandomIndex([]int{100 - *alias.Weight, *alias.Weight})
	if idx == 0 {
		return alias.Version1, nil
	}
	return *alias.Version2, nil
}

// List retrieves all agent configs for a project
func (s *AgentConfigService) List(ctx context.Context, projectID uuid.UUID) ([]*AgentConfigSummary, error) {
	configs, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent configs: %w", err)
	}

	return configs, nil
}

// ListVersions retrieves all versions of an agent config by agent_id
func (s *AgentConfigService) ListVersions(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) ([]*AgentConfig, error) {
	configs, err := s.repo.ListVersions(ctx, projectID, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent config versions: %w", err)
	}

	return configs, nil
}

// ListVersionsByName retrieves all versions of an agent config by name (for backward compatibility)
func (s *AgentConfigService) ListVersionsByName(ctx context.Context, projectID uuid.UUID, name string) ([]*AgentConfig, error) {
	configs, err := s.repo.ListVersionsByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent config versions: %w", err)
	}

	return configs, nil
}

// Delete deletes all versions of an agent config by agent_id
func (s *AgentConfigService) Delete(ctx context.Context, agentID uuid.UUID) error {
	err := s.repo.Delete(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to delete agent config: %w", err)
	}

	return nil
}

// DeleteByName deletes all versions of an agent config by name
func (s *AgentConfigService) DeleteByName(ctx context.Context, projectID uuid.UUID, name string) error {
	err := s.repo.DeleteByName(ctx, projectID, name)
	if err != nil {
		return fmt.Errorf("failed to delete agent config: %w", err)
	}

	return nil
}

// DeleteVersion deletes a specific version of an agent config
func (s *AgentConfigService) DeleteVersion(ctx context.Context, agentID uuid.UUID, version int) error {
	// Prevent deletion of version 0
	if version == 0 {
		return fmt.Errorf("cannot delete version 0")
	}

	err := s.repo.DeleteVersion(ctx, agentID, version)
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

// CreateAlias creates a new alias for an agent config
func (s *AgentConfigService) CreateAlias(ctx context.Context, projectID uuid.UUID, agentName string, req *CreateAliasRequest) (*AgentConfigAlias, error) {
	// Get agent_id from name
	agentID, err := s.repo.GetAgentIDByName(ctx, projectID, agentName)
	if err != nil {
		return nil, err
	}

	// Validate that version1 is provided
	if req.Version1 == 0 {
		return nil, fmt.Errorf("version1 is required")
	}

	// Validate weight if version2 is provided
	if req.Version2 != nil && req.Weight == nil {
		return nil, fmt.Errorf("weight is required when version2 is set")
	}

	return s.repo.CreateAlias(ctx, projectID, agentID, req)
}

// CreateAliasByAgentID creates a new alias for an agent config by agent_id
func (s *AgentConfigService) CreateAliasByAgentID(ctx context.Context, projectID, agentID uuid.UUID, req *CreateAliasRequest) (*AgentConfigAlias, error) {
	// Validate that version1 is provided
	if req.Version1 == 0 {
		return nil, fmt.Errorf("version1 is required")
	}

	// Validate weight if version2 is provided
	if req.Version2 != nil && req.Weight == nil {
		return nil, fmt.Errorf("weight is required when version2 is set")
	}

	return s.repo.CreateAlias(ctx, projectID, agentID, req)
}

// GetAlias retrieves an alias by ID
func (s *AgentConfigService) GetAlias(ctx context.Context, projectID, id uuid.UUID) (*AgentConfigAlias, error) {
	return s.repo.GetAlias(ctx, projectID, id)
}

// GetAliasByName retrieves an alias by agent name and alias name
func (s *AgentConfigService) GetAliasByName(ctx context.Context, projectID uuid.UUID, agentName, aliasName string) (*AgentConfigAlias, error) {
	agentID, err := s.repo.GetAgentIDByName(ctx, projectID, agentName)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAliasByName(ctx, projectID, agentID, aliasName)
}

// ListAliases lists all aliases for an agent by name
func (s *AgentConfigService) ListAliases(ctx context.Context, projectID uuid.UUID, agentName string) ([]*AgentConfigAlias, error) {
	agentID, err := s.repo.GetAgentIDByName(ctx, projectID, agentName)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAliases(ctx, projectID, agentID)
}

// ListAliasesByAgentID lists all aliases for an agent by agent_id
func (s *AgentConfigService) ListAliasesByAgentID(ctx context.Context, projectID, agentID uuid.UUID) ([]*AgentConfigAlias, error) {
	return s.repo.ListAliases(ctx, projectID, agentID)
}

// UpdateAlias updates an existing alias
func (s *AgentConfigService) UpdateAlias(ctx context.Context, projectID, id uuid.UUID, req *UpdateAliasRequest) (*AgentConfigAlias, error) {
	return s.repo.UpdateAlias(ctx, projectID, id, req)
}

// DeleteAlias deletes an alias by ID
func (s *AgentConfigService) DeleteAlias(ctx context.Context, projectID, id uuid.UUID) error {
	return s.repo.DeleteAlias(ctx, projectID, id)
}
