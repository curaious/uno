package tools

import (
	"bytes"
	"context"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/curaious/uno/pkg/sandbox"
	"github.com/curaious/uno/pkg/sandbox/sandbox_daemon"
)

type SandboxTool struct {
	*core.BaseTool
	svc   sandbox.Manager
	image string
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
		svc:   svc,
		image: image,
	}
}

func (t *SandboxTool) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	var in Input
	err := sonic.Unmarshal([]byte(params.Arguments), &in)
	if err != nil {
		return nil, err
	}

	sb, err := t.svc.CreateSandbox(ctx, t.image, params.AgentName, params.Namespace, params.ConversationID)
	if err != nil {
		return nil, err
	}

	req := sandbox_daemon.ExecRequest{
		Command:        in.Code,
		Args:           nil,
		Script:         "",
		TimeoutSeconds: 0,
		Workdir:        "",
		Env:            nil,
	}

	b, _ := sonic.Marshal(req)

	resp, err := http.DefaultClient.Post("http://"+sb.PodIP+":8080/exec/bash", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	var res map[string]any
	utils.DecodeJSON(resp.Body, &res)
	txt, _ := sonic.Marshal(res)

	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr(string(txt)),
		},
	}, nil
}
