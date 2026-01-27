package restate_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
)

type RestateTool struct {
	restateCtx  restate.WorkflowContext
	wrappedTool core.Tool
}

func NewRestateTool(restateCtx restate.WorkflowContext, wrappedTool core.Tool) *RestateTool {
	return &RestateTool{
		restateCtx:  restateCtx,
		wrappedTool: wrappedTool,
	}
}

func (t *RestateTool) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	return restate.Run(t.restateCtx, func(ctx restate.RunContext) (*responses.FunctionCallOutputMessage, error) {
		return t.wrappedTool.Execute(ctx, params)
	}, restate.WithName(params.Name+"_ToolCall"))
}

func (t *RestateTool) Tool(ctx context.Context) *responses.ToolUnion {
	return t.wrappedTool.Tool(ctx)
}

func (t *RestateTool) NeedApproval() bool {
	return t.wrappedTool.NeedApproval()
}
