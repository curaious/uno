package adapters

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/integrations"
	"github.com/praveen001/uno/internal/services/prompt"
	"github.com/praveen001/uno/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ExternalPromptPersistence struct {
	Endpoint  string
	projectID uuid.UUID
	name      string
	label     string
}

func NewExternalPromptPersistence(endpoint string, projectID uuid.UUID, name string, label string) *ExternalPromptPersistence {
	return &ExternalPromptPersistence{
		Endpoint:  endpoint,
		projectID: projectID,
		name:      name,
		label:     label,
	}
}

func (p *ExternalPromptPersistence) GetPrompt(ctx context.Context) (string, error) {
	// Read the prompt from file
	url := fmt.Sprintf("%s/api/agent-server/prompts/%s/label/%s?project_id=%s", p.Endpoint, p.name, p.label, p.projectID)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data := Response[prompt.PromptVersionWithPrompt]{}
	if err := utils.DecodeJSON(resp.Body, &data); err != nil {
		return "", err
	}

	return data.Data.Template, nil
}

type LangfusePromptPersistence struct {
	client *integrations.LangfuseClient
	name   string
	label  string
}

func NewLangfusePromptProvider(endpoint, username, password string, name string, label string) *LangfusePromptPersistence {
	client := integrations.NewLangfuseClient(endpoint, username, password)

	return &LangfusePromptPersistence{
		client: client,
		name:   name,
		label:  label,
	}
}

func (p *LangfusePromptPersistence) GetPrompt(ctx context.Context) (string, error) {
	// Get the prompt from Langfuse
	promptResp, err := p.client.GetPrompt(p.name, p.label)
	if err != nil {
		return "", err
	}

	// Traces
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("langfuse.observation.prompt.name", promptResp.Name))
	span.SetAttributes(attribute.Int("langfuse.observation.prompt.version", promptResp.Version))

	return promptResp.Prompt, nil
}
