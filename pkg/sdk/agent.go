package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type AgentOptions struct {
	Name        string
	LLM         llm.Provider
	Tools       []core.Tool
	Output      map[string]any
	History     core.ChatHistory
	Parameters  responses.Parameters
	Instruction core.SystemPromptProvider
}

func (c *SDK) NewAgent(options *AgentOptions) *agents.Agent {
	return agents.NewAgent(&agents.AgentOptions{
		Name:        options.Name,
		LLM:         options.LLM,
		History:     options.History,
		Parameters:  options.Parameters,
		Output:      options.Output,
		Tools:       options.Tools,
		Instruction: options.Instruction,
	})
}
