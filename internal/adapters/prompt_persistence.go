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
	name      string
	label     string
}

func NewInternalPromptPersistence(svc *prompt.PromptService, projectID uuid.UUID, name string, label string) *InternalPromptPersistence {
	return &InternalPromptPersistence{
		svc:       svc,
		projectID: projectID,
		name:      name,
		label:     label,
	}
}

func (p *InternalPromptPersistence) LoadPrompt(ctx context.Context) (string, error) {
	// Get prompt version by label
	version, err := p.svc.GetPromptVersionByLabel(ctx, p.projectID, p.name, p.label)
	if err != nil {
		version, err = p.svc.GetLatestPromptVersion(ctx, p.projectID, p.name)
		if err != nil {
			return "", err
		}
	}

	return version.Template, nil
}
