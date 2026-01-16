package adapters

import (
	"context"

	"github.com/curaious/uno/internal/services/prompt"
	"github.com/google/uuid"
)

// InternalPromptPersistence implements core.SystemPromptProvider using internal services
type InternalPromptPersistence struct {
	svc       *prompt.PromptService
	projectID uuid.UUID
	promptID  uuid.UUID
	version   int
}

func NewInternalPromptPersistence(svc *prompt.PromptService, projectID uuid.UUID, promptID uuid.UUID, version int) *InternalPromptPersistence {
	return &InternalPromptPersistence{
		svc:       svc,
		projectID: projectID,
		promptID:  promptID,
		version:   version,
	}
}

func (p *InternalPromptPersistence) LoadPrompt(ctx context.Context) (string, error) {
	// Get prompt version by label
	version, err := p.svc.GetPromptVersionByVersion(ctx, p.projectID, p.promptID, p.version)
	if err != nil {
		return "", err
	}

	return version.Template, nil
}
