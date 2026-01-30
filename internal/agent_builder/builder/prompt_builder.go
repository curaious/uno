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

func BuildPrompt(svc *prompt.PromptService, projectID uuid.UUID, config *agent_config.PromptConfig, skillConfig []agent_config.SkillConfig) core.SystemPromptProvider {
	var opts []prompts.PromptOption
	var skills []core.Skill
	for _, skill := range skillConfig {
		skills = append(skills, core.Skill{
			Name:         skill.Name,
			Description:  skill.Description,
			FileLocation: skill.FileLocation,
		})
	}

	if skills != nil && len(skills) > 0 {
		opts = append(opts, prompts.WithSkills(skills))
	}

	var instruction core.SystemPromptProvider
	if config.RawPrompt != nil {
		instruction = prompts.New(*config.RawPrompt, opts...)
	} else if *config.PromptID != uuid.Nil {
		instruction = prompts.NewWithLoader(adapters.NewInternalPromptPersistence(svc, projectID, *config.PromptID, *config.Version), opts...)
	} else {
		slog.Warn("no system prompt provided")
	}

	return instruction
}
