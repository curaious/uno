package tools

import (
	"context"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type AgentTool struct {
	*responses.ToolUnion
	agent *agents.Agent
}

func NewAgentTool(t *responses.ToolUnion, agent *agents.Agent) *AgentTool {
	return &AgentTool{
		ToolUnion: t,
		agent:     agent,
	}
}

func (t *AgentTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	output, err := t.agent.Execute(ctx,
		[]responses.InputMessageUnion{
			{
				OfEasyInput: &responses.EasyMessage{
					Content: responses.EasyInputContentUnion{OfString: &params.Arguments},
				},
			},
		},
		core.NilCallback,
	)
	if err != nil {
		return nil, err
	}

	data := ""
	for _, out := range output {
		for _, content := range out.OfOutputMessage.Content {
			if content.OfOutputText != nil {
				data += content.OfOutputText.Text
			}
		}
	}

	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr(data),
		},
	}, nil
}
