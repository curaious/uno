package core

import (
	"context"
	"text/template"

	"github.com/praveen001/uno/pkg/llm/responses"
)

type SystemPromptProvider interface {
	GetPrompt(ctx context.Context, msgs []responses.InputMessageUnion) (string, error)
}

type SystemPromptResolver func(tmpl *template.Template, msgs []responses.InputMessageUnion) (string, error)
