package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func (c *SDK) Prompt(prompt string) *prompts.SimplePrompt {
	return prompts.New(prompt)
}

func (c *SDK) RemotePrompt(name, label string) *prompts.SimplePrompt {
	return prompts.NewWithLoader(adapters.NewExternalPromptPersistence(c.endpoint, c.projectId, name, label))
}

func (c *SDK) CustomPrompt(loader prompts.PromptLoader) *prompts.SimplePrompt {
	return prompts.NewWithLoader(loader)
}
