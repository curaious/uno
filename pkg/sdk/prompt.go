package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func (c *SDK) NewPrompt(prompt string, resolvers ...prompts.PromptResolverFn) *prompts.SimplePrompt {
	return prompts.New(prompt, resolvers...)
}

func (c *SDK) NewRemotePrompt(name, label string, resolvers ...prompts.PromptResolverFn) *prompts.SimplePrompt {
	return prompts.NewWithLoader(adapters.NewExternalPromptPersistence(c.endpoint, c.projectId, name, label), resolvers...)
}

func (c *SDK) NewCustomPrompt(loader prompts.PromptLoader, resolvers ...prompts.PromptResolverFn) *prompts.SimplePrompt {
	return prompts.NewWithLoader(loader, resolvers...)
}
