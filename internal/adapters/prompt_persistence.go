package adapters

import (
	"context"

	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services/prompt"
)

// InternalPromptPersistence implements core.SystemPromptProvider using internal services
type InternalPromptPersistence struct {
	svc       *prompt.PromptService
	projectID uuid.UUID
}

func NewInternalPromptPersistence(svc *prompt.PromptService, projectID uuid.UUID) *InternalPromptPersistence {
	return &InternalPromptPersistence{
		svc:       svc,
		projectID: projectID,
	}
}

func (p *InternalPromptPersistence) GetPrompt(ctx context.Context, name string, label string) (string, error) {
	// Get prompt version by label
	version, err := p.svc.GetPromptVersionByLabel(ctx, p.projectID, name, label)
	if err != nil {
		version, err = p.svc.GetLatestPromptVersion(ctx, p.projectID, name)
		if err != nil {
			return "", err
		}
	}

	return version.Template, nil
}
