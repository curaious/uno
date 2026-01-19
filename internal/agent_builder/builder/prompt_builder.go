package builder

import (
	"log/slog"

	"github.com/curaious/uno/internal/adapters"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/internal/services/prompt"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/prompts"
	"github.com/google/uuid"
)

func BuildPrompt(svc *prompt.PromptService, projectID uuid.UUID, config *agent_config.PromptConfig) core.SystemPromptProvider {
	var instruction core.SystemPromptProvider
	if config.RawPrompt != nil {
		instruction = prompts.New(*config.RawPrompt)
	} else if *config.PromptID != uuid.Nil {
		instruction = prompts.NewWithLoader(adapters.NewInternalPromptPersistence(svc, projectID, *config.PromptID, *config.Version))
	} else {
		slog.Warn("no system prompt provided")
	}

	return instruction
}
