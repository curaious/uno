package prompt

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// PromptService handles business logic for prompts and prompt versions
type PromptService struct {
	repo *PromptRepo
}

// NewPromptService creates a new prompt service
func NewPromptService(repo *PromptRepo) *PromptService {
	return &PromptService{repo: repo}
}

// validateHandlebarsTemplate performs basic validation of Handlebars template syntax
func (s *PromptService) validateHandlebarsTemplate(template string) error {
	// Check for balanced braces
	openBraces := strings.Count(template, "{{")
	closeBraces := strings.Count(template, "}}")

	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces in template")
	}

	// Check for valid variable syntax using regex
	// Valid: {{variable_name}}, {{variable-name}}, {{variable123}}
	// Invalid: {{}}, {{ variable }}, {{variable name}}
	handlebarsRegex := regexp.MustCompile(`\{\{[^}]*\}\}`)
	matches := handlebarsRegex.FindAllString(template, -1)

	for _, match := range matches {
		// Remove the braces to get the variable name
		variableName := strings.TrimSpace(match[2 : len(match)-2])

		// Check if variable name is empty or contains invalid characters
		if variableName == "" {
			return fmt.Errorf("empty variable name in template")
		}

		// Check for valid variable name (alphanumeric, underscore, hyphen)
		validVarRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
		if !validVarRegex.MatchString(variableName) {
			return fmt.Errorf("invalid variable name '%s' in template", variableName)
		}
	}

	return nil
}

// CreatePrompt creates a new prompt with its first version
func (s *PromptService) CreatePrompt(ctx context.Context, projectID uuid.UUID, req *CreatePromptRequest) (*PromptWithLatestVersion, error) {
	// Check if prompt with same name already exists
	existing, err := s.repo.GetPromptByName(ctx, projectID, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("prompt with name '%s' already exists", req.Name)
	}

	// Validate template syntax
	if err := s.validateHandlebarsTemplate(req.Template); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Create the prompt
	prompt, err := s.repo.CreatePrompt(ctx, projectID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	// Create the first version
	version, err := s.repo.CreatePromptVersion(ctx, projectID, prompt.ID, req.Template, req.CommitMessage, req.Label)
	if err != nil {
		return nil, fmt.Errorf("failed to create first prompt version: %w", err)
	}

	// Return prompt with latest version info
	return &PromptWithLatestVersion{
		Prompt:          *prompt,
		LatestVersion:   &version.Version,
		LatestCommitMsg: &version.CommitMessage,
		LatestLabel:     version.Label,
	}, nil
}

// CreatePromptVersion creates a new prompt version
func (s *PromptService) CreatePromptVersion(ctx context.Context, projectID uuid.UUID, promptName string, req *CreatePromptVersionRequest) (*PromptVersionWithPrompt, error) {
	// Get the prompt by name
	prompt, err := s.repo.GetPromptByName(ctx, projectID, promptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	// Validate template syntax
	if err := s.validateHandlebarsTemplate(req.Template); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Create the version
	version, err := s.repo.CreatePromptVersion(ctx, projectID, prompt.ID, req.Template, req.CommitMessage, req.Label)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt version: %w", err)
	}

	// Return with prompt name
	versionWithPrompt := &PromptVersionWithPrompt{
		PromptVersion: *version,
		PromptName:    prompt.Name,
	}

	return versionWithPrompt, nil
}

// GetPrompt retrieves a prompt by name (returns with latest version info)
func (s *PromptService) GetPrompt(ctx context.Context, projectID uuid.UUID, promptName string) (*PromptWithLatestVersion, error) {
	// Get the prompt
	prompt, err := s.repo.GetPromptByName(ctx, projectID, promptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	// Get latest version info
	latestVersion, err := s.repo.GetLatestPromptVersion(ctx, projectID, promptName)
	if err != nil {
		// If no versions exist, return prompt without version info
		return &PromptWithLatestVersion{
			Prompt: *prompt,
		}, nil
	}

	return &PromptWithLatestVersion{
		Prompt:          *prompt,
		LatestVersion:   &latestVersion.Version,
		LatestCommitMsg: &latestVersion.CommitMessage,
		LatestLabel:     latestVersion.Label,
	}, nil
}

// GetPromptVersion retrieves a specific prompt version
func (s *PromptService) GetPromptVersion(ctx context.Context, projectID uuid.UUID, promptName string, version int) (*PromptVersionWithPrompt, error) {
	versionWithPrompt, err := s.repo.GetPromptVersion(ctx, projectID, promptName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	return versionWithPrompt, nil
}

// GetPromptVersionByLabel retrieves a prompt version by label
func (s *PromptService) GetPromptVersionByLabel(ctx context.Context, projectID uuid.UUID, promptName, label string) (*PromptVersionWithPrompt, error) {
	versionWithPrompt, err := s.repo.GetPromptVersionByLabel(ctx, projectID, promptName, label)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt version by label: %w", err)
	}

	return versionWithPrompt, nil
}

// GetLatestPromptVersion retrieves the latest version of a prompt by name
func (s *PromptService) GetLatestPromptVersion(ctx context.Context, projectID uuid.UUID, promptName string) (*PromptVersionWithPrompt, error) {
	versionWithPrompt, err := s.repo.GetLatestPromptVersion(ctx, projectID, promptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest prompt version: %w", err)
	}

	return versionWithPrompt, nil
}

// ListPrompts retrieves all prompts with their latest version information
func (s *PromptService) ListPrompts(ctx context.Context, projectID uuid.UUID) ([]*PromptWithLatestVersion, error) {
	prompts, err := s.repo.ListPrompts(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return prompts, nil
}

// ListPromptVersions retrieves all versions for a specific prompt
func (s *PromptService) ListPromptVersions(ctx context.Context, projectID uuid.UUID, promptName string) ([]*PromptVersion, error) {
	// Get the prompt to get its ID
	prompt, err := s.repo.GetPromptByName(ctx, projectID, promptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	versions, err := s.repo.ListPromptVersions(ctx, projectID, prompt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompt versions: %w", err)
	}

	return versions, nil
}

// UpdatePromptVersionLabel updates the label of a prompt version
func (s *PromptService) UpdatePromptVersionLabel(ctx context.Context, projectID uuid.UUID, promptName string, version int, req *UpdatePromptVersionLabelRequest) (*PromptVersion, error) {
	// Get the version to get its ID
	versionWithPrompt, err := s.repo.GetPromptVersion(ctx, projectID, promptName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt version: %w", err)
	}

	// Update the label
	updatedVersion, err := s.repo.UpdatePromptVersionLabel(ctx, projectID, versionWithPrompt.ID, req.Label)
	if err != nil {
		return nil, fmt.Errorf("failed to update prompt version label: %w", err)
	}

	return updatedVersion, nil
}

// DeletePromptVersion deletes a specific prompt version
func (s *PromptService) DeletePromptVersion(ctx context.Context, projectID uuid.UUID, promptName string, version int) error {
	// Get the version to get its ID
	versionWithPrompt, err := s.repo.GetPromptVersion(ctx, projectID, promptName, version)
	if err != nil {
		return fmt.Errorf("failed to get prompt version: %w", err)
	}

	err = s.repo.DeletePromptVersion(ctx, versionWithPrompt.ID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt version: %w", err)
	}

	return nil
}

// DeletePrompt deletes a prompt and all its versions
func (s *PromptService) DeletePrompt(ctx context.Context, projectID uuid.UUID, promptName string) error {
	// Get the prompt to get its ID
	prompt, err := s.repo.GetPromptByName(ctx, projectID, promptName)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	err = s.repo.DeletePrompt(ctx, projectID, prompt.ID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	return nil
}
