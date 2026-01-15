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

// InternalPromptPersistenceV2 implements core.SystemPromptProvider using internal services
type InternalPromptPersistenceV2 struct {
	svc       *prompt.PromptService
	projectID uuid.UUID
	promptID  uuid.UUID
}

func NewInternalPromptPersistenceV2(svc *prompt.PromptService, projectID uuid.UUID, promptID uuid.UUID) *InternalPromptPersistenceV2 {
	return &InternalPromptPersistenceV2{
		svc:       svc,
		projectID: projectID,
		promptID:  promptID,
	}
}

func (p *InternalPromptPersistenceV2) LoadPrompt(ctx context.Context) (string, error) {
	// Get prompt version by label
	version, err := p.svc.GetPromptVersionByID(ctx, p.projectID, p.promptID)
	if err != nil {
		return "", err
	}

	return version.Template, nil
}
