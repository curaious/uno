package tools

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
)

type CodeExecutionTool struct {
	*core.BaseTool
}

func NewCodeExecutionTool() *CodeExecutionTool {
	return &CodeExecutionTool{}
}

func (t *CodeExecutionTool) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	return nil, nil
}

func (t *CodeExecutionTool) Tool(ctx context.Context) *responses.ToolUnion {
	return &responses.ToolUnion{OfCodeExecution: &responses.CodeExecutionTool{
		Container: &responses.CodeExecutionToolContainerUnion{
			ContainerConfig: &responses.CodeExecutionToolContainerConfig{
				Type:        "auto",
				MemoryLimit: "4g",
			},
		},
	}}
}
