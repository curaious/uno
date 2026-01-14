package prompts

import (
	"context"
	"regexp"
	"text/template"

	"github.com/curaious/uno/internal/utils"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("PromptManager")

type PromptLoader interface {
	// LoadPrompt loads the prompt from the source and returns it as string
	LoadPrompt(ctx context.Context) (string, error)
}

type PromptResolverFn func(string, map[string]any) (string, error)

type StringLoader struct {
	String string
}

func NewStringLoader(str string) *StringLoader {
	return &StringLoader{
		String: str,
	}
}

func (sl *StringLoader) LoadPrompt(ctx context.Context) (string, error) {
	return sl.String, nil
}

type SimplePrompt struct {
	loader   PromptLoader
	resolver PromptResolverFn
}

func New(prompt string, opts ...PromptOption) *SimplePrompt {
	return NewWithLoader(NewStringLoader(prompt), opts...)
}

func NewWithLoader(loader PromptLoader, opts ...PromptOption) *SimplePrompt {
	sp := &SimplePrompt{
		loader:   loader,
		resolver: DefaultResolver,
	}

	for _, op := range opts {
		op(sp)
	}

	return sp
}

type PromptOption func(*SimplePrompt)

func WithResolver(resolverFn PromptResolverFn) PromptOption {
	return func(sp *SimplePrompt) {
		sp.resolver = resolverFn
	}
}

func (sp *SimplePrompt) GetPrompt(ctx context.Context, data map[string]any) (string, error) {
	ctx, span := tracer.Start(ctx, "GetPrompt")
	defer span.End()

	prompt, err := sp.loader.LoadPrompt(ctx)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	if data == nil {
		return prompt, nil
	}

	return sp.resolver(prompt, data)
}

func stringToTemplate(promptStr string) (*template.Template, error) {
	re := regexp.MustCompile(`{{(\w.+)}}`)
	promptStr = re.ReplaceAllString(promptStr, "{{ .$1 }}")

	return template.New("file_prompt").Parse(promptStr)
}

func DefaultResolver(prompt string, data map[string]any) (string, error) {
	tmpl, err := stringToTemplate(prompt)
	if err != nil {
		return prompt, err
	}

	return utils.ExecuteTemplate(tmpl, data)
}
