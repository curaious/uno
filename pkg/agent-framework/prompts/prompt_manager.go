package prompts

import (
	"context"
	"fmt"
	"regexp"
	"text/template"

	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var promptTracer = otel.Tracer("PromptManager")

type PromptPersistence interface {
	// GetPrompt returns prompt string for the given a prompt name, and associated label.
	GetPrompt(ctx context.Context, name string, label string) (string, error)
}

type PromptManager struct {
	PromptPersistence

	name     string
	label    string
	resolver core.SystemPromptResolver
}

// NewPromptManager creates a new PromptManager instance.
// It uses the given prompt persistence to fetch prompts by name and label.
// The resolver is used to resolve the prompt template with the provided messages.
func NewPromptManager(persistence PromptPersistence, name string, label string, resolver core.SystemPromptResolver) *PromptManager {
	return &PromptManager{
		PromptPersistence: persistence,

		name:     name,
		label:    label,
		resolver: resolver,
	}
}

func (pm *PromptManager) GetPrompt(ctx context.Context, msgs []responses.InputMessageUnion) (string, error) {
	ctx, span := promptTracer.Start(ctx, "PromptManager.GetPrompt")
	defer span.End()

	span.SetAttributes(
		attribute.String("prompt.name", pm.name),
		attribute.String("prompt.label", pm.label),
	)

	// If prompt persistence is not provided, we cannot proceed.
	if pm.PromptPersistence == nil {
		err := fmt.Errorf("prompt persistence not initialized")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// Fetch the prompt template
	promptStr, err := pm.PromptPersistence.GetPrompt(ctx, pm.name, pm.label)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// If resolver is provided, resolve the prompt template with messages
	if pm.resolver != nil {
		re := regexp.MustCompile(`{{(\w.+)}}`)
		promptStr = re.ReplaceAllString(promptStr, "{{ .$1 }}")

		tmpl, err := template.New("file_prompt").Parse(promptStr)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return "", err
		}

		return pm.resolver(tmpl, msgs)
	}

	return promptStr, nil
}
