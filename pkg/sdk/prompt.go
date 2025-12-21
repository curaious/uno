package sdk

import (
	"context"
	"text/template"

	"github.com/praveen001/uno/internal/utils"
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
