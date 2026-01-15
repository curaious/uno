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
}

func NewInternalPromptPersistence(svc *prompt.PromptService, projectID uuid.UUID, promptID uuid.UUID) *InternalPromptPersistence {
	return &InternalPromptPersistence{
		svc:       svc,
		projectID: projectID,
		promptID:  promptID,
	}
}

func (p *InternalPromptPersistence) LoadPrompt(ctx context.Context) (string, error) {
	// Get prompt version by label
	version, err := p.svc.GetPromptVersionByID(ctx, p.projectID, p.promptID)
	if err != nil {
		return "", err
	}

	return version.Template, nil
}
