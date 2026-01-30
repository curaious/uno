package tools

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/curaious/uno/pkg/sandbox"
	"github.com/curaious/uno/pkg/sandbox/daemon"
)

type SandboxTool struct {
	*core.BaseTool
	sandboxManager sandbox.Manager
	image          string
}

type Input struct {
	Code string `json:"code"`
}

func NewSandboxTool(svc sandbox.Manager, image string) *SandboxTool {
	return &SandboxTool{
		BaseTool: &core.BaseTool{
			ToolUnion: responses.ToolUnion{
				OfFunction: &responses.FunctionTool{
					Name:        "execute_bash_commands",
					Description: utils.Ptr("Execute bash command and get the output"),
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"code": map[string]any{
								"type":        "string",
								"description": "bash command to be executed",
							},
						},
						"required": []string{"code"},
					},
				},
			},
			RequiresApproval: false,
		},
		sandboxManager: svc,
		image:          image,
	}
}

func (t *SandboxTool) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	var in Input
	err := sonic.Unmarshal([]byte(params.Arguments), &in)
	if err != nil {
		return nil, err
	}

	sb, err := t.sandboxManager.CreateSandbox(ctx, t.image, params.AgentName, params.Namespace, params.ConversationID)
	if err != nil {
		return nil, err
	}

	// Create a sandbox daemon client
	cli := daemon.NewClient(sb)

	// Run bash command
	res, err := cli.RunBashCommand(ctx, &daemon.ExecRequest{
		Command:        in.Code,
		Args:           nil,
		Script:         "",
		TimeoutSeconds: 0,
		Workdir:        "",
		Env:            nil,
	})
	if err != nil {
		return nil, err
	}

	// Serialize the output
	txt, _ := sonic.Marshal(res)

	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr(string(txt)),
		},
	}, nil
}
