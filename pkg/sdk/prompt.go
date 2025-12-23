package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func (c *SDK) Prompt(prompt string, resolvers ...prompts.PromptResolverFn) *prompts.SimplePrompt {
	return prompts.New(prompt, resolvers...)
}

func (c *SDK) RemotePrompt(name, label string, resolvers ...prompts.PromptResolverFn) *prompts.SimplePrompt {
	return prompts.NewWithLoader(adapters.NewExternalPromptPersistence(c.endpoint, c.projectId, name, label), resolvers...)
}

func (c *SDK) CustomPrompt(loader prompts.PromptLoader, resolvers ...prompts.PromptResolverFn) *prompts.SimplePrompt {
	return prompts.NewWithLoader(loader, resolvers...)
}
