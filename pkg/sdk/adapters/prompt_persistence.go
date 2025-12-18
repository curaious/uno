package adapters

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
}

func NewExternalPromptPersistence(endpoint string, projectID uuid.UUID) *ExternalPromptPersistence {
	return &ExternalPromptPersistence{
		Endpoint:  endpoint,
		projectID: projectID,
	}
}

func (p *ExternalPromptPersistence) GetPrompt(ctx context.Context, name string, label string) (string, error) {
	// Read the prompt from file
	url := fmt.Sprintf("%s/api/agent-server/prompts/%s/label/%s?project_id=%s", p.Endpoint, name, label, p.projectID)

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
}

func NewLangfusePromptProvider(endpoint, username, password string) *LangfusePromptPersistence {
	client := integrations.NewLangfuseClient(endpoint, username, password)

	return &LangfusePromptPersistence{
		client: client,
	}
}

func (p *LangfusePromptPersistence) GetPrompt(ctx context.Context, name string, label string) (string, error) {
	// Get the prompt from Langfuse
	promptResp, err := p.client.GetPrompt(name, label)
	if err != nil {
		return "", err
	}

	// Traces
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("langfuse.observation.prompt.name", promptResp.Name))
	span.SetAttributes(attribute.Int("langfuse.observation.prompt.version", promptResp.Version))

	return promptResp.Prompt, nil
}

type FilePromptPersistence struct {
}

func NewFilePromptPersistence() *FilePromptPersistence {
	return &FilePromptPersistence{}
}

func (p *FilePromptPersistence) GetPrompt(ctx context.Context, name string, label string) (string, error) {
	// Read the prompt from file
	promptBytes, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}

	return string(promptBytes), nil
}
