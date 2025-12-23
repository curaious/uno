package prompts

import (
	"context"
	"regexp"
	"text/template"

	"github.com/praveen001/uno/internal/utils"
	"go.opentelemetry.io/otel"
)

var promptTracer = otel.Tracer("PromptManager")

type PromptLoader interface {
	// GetPrompt returns loads the prompt from the source and returns it as string
	GetPrompt(ctx context.Context) (string, error)
}

type StringLoader struct {
	String string
}

func NewStringLoader(str string) *StringLoader {
	return &StringLoader{
		String: str,
	}
}

func (sl *StringLoader) GetPrompt(ctx context.Context) (string, error) {
	return sl.String, nil
}

type SimplePrompt struct {
	loader    PromptLoader
	Resolvers []PromptResolverFn
}

func New(prompt string, resolvers ...PromptResolverFn) *SimplePrompt {
	return NewWithLoader(NewStringLoader(prompt), resolvers...)
}

func NewWithLoader(loader PromptLoader, resolvers ...PromptResolverFn) *SimplePrompt {
	return &SimplePrompt{
		loader:    loader,
		Resolvers: resolvers,
	}
}

func (sp *SimplePrompt) WithResolver(fn PromptResolverFn) *SimplePrompt {
	sp.Resolvers = append(sp.Resolvers, fn)
	return sp
}

func (sp *SimplePrompt) WithDefaultResolver(data map[string]any) *SimplePrompt {
	sp.Resolvers = append(sp.Resolvers, NewDefaultResolver(data))
	return sp
}

func (sp *SimplePrompt) GetPrompt(ctx context.Context) (string, error) {
	promptStr, err := sp.loader.GetPrompt(ctx)
	if err != nil {
		return "", err
	}

	if sp.Resolvers != nil {
		var err error
		for _, resolver := range sp.Resolvers {
			promptStr, err = resolver(promptStr)
			if err != nil {
				return "", err
			}
		}
	}

	return promptStr, nil
}

type PromptResolver func(*SimplePrompt)

type PromptResolverFn func(string) (string, error)

func WithResolver(resolver PromptResolverFn) PromptResolverFn {
	return resolver
}

func WithDefaultResolver(data map[string]any) PromptResolverFn {
	return NewDefaultResolver(data)
}

type DefaultResolver struct{}

func NewDefaultResolver(data map[string]any) PromptResolverFn {
	return func(s string) (string, error) {
		tmpl, err := stringToTemplate(s)
		if err != nil {
			return "", err
		}

		return utils.ExecuteTemplate(tmpl, data)
	}
}

func stringToTemplate(promptStr string) (*template.Template, error) {
	re := regexp.MustCompile(`{{(\w.+)}}`)
	promptStr = re.ReplaceAllString(promptStr, "{{ .$1 }}")

	return template.New("file_prompt").Parse(promptStr)
}
