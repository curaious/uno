package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/workflow"
)

type TemporalTool struct {
	wrappedTool core.Tool
}

func NewTemporalTool(wrappedTool core.Tool) *TemporalTool {
	return &TemporalTool{
		wrappedTool: wrappedTool,
	}
}

func (t *TemporalTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return t.wrappedTool.Execute(ctx, params)
}

type TemporalToolProxy struct {
	workflowCtx workflow.Context
	prefix      string
	wrappedTool core.Tool
}

func NewTemporalToolProxy(workflowCtx workflow.Context, prefix string, wrappedTool core.Tool) core.Tool {
	return &TemporalToolProxy{
		workflowCtx: workflowCtx,
		prefix:      prefix,
		wrappedTool: wrappedTool,
	}
}

func (t *TemporalToolProxy) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	var output *responses.FunctionCallOutputMessage
	err := workflow.ExecuteActivity(t.workflowCtx, t.prefix+"_ExecuteToolActivity", params).Get(t.workflowCtx, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (t *TemporalToolProxy) Tool(ctx context.Context) *responses.ToolUnion {
	return t.wrappedTool.Tool(ctx)
}

func (t *TemporalToolProxy) NeedApproval() bool {
	return t.wrappedTool.NeedApproval()
}
