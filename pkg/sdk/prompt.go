package sdk

import (
	"context"
	"text/template"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

type PromptTemplate struct {
	Template string
}

func NewPromptTemplate(tmpl string) *PromptTemplate {
	return &PromptTemplate{
		Template: tmpl,
	}
}

func (p *PromptTemplate) Execute(ctx context.Context, data map[string]any) (string, error) {
	tmpl, err := template.New("file_prompt").Parse(p.Template)
	if err != nil {
		return "", err
	}

	return utils.ExecuteTemplate(tmpl, data)
}

func (c *SDK) NewPromptManager(name string, label string, resolver core.SystemPromptResolver) core.SystemPromptProvider {
	return prompts.NewPromptManager(
		adapters.NewExternalPromptPersistence(c.endpoint, c.projectId),
		name,
		label,
		resolver,
	)
}
