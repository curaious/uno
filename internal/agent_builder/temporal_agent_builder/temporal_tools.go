package temporal_agent_builder

import (
	"context"
	"os"

	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/tools"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/workflow"
)

func (b *AgentBuilder) SandboxTool(ctx context.Context, image string, toolCall *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	sandboxTool := tools.NewSandboxTool(b.sandboxManager, image)
	return sandboxTool.Execute(ctx, toolCall)
}

type TemporalToolProxy struct {
	workflowCtx workflow.Context
	image       string
	wrappedTool *tools.SandboxTool
}

func NewTemporalSandboxToolProxy(workflowCtx workflow.Context, image string) *TemporalToolProxy {
	return &TemporalToolProxy{
		workflowCtx: workflowCtx,
		image:       image,
		wrappedTool: tools.NewSandboxTool(nil, image),
	}
}

func (p *TemporalToolProxy) Execute(ctx context.Context, toolCall *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	var out responses.FunctionCallOutputMessage
	err := workflow.ExecuteActivity(p.workflowCtx, "SandboxTool", p.image, toolCall).Get(p.workflowCtx, &out)
	return &out, err
}

func (t *TemporalToolProxy) Tool(ctx context.Context) *responses.ToolUnion {
	return t.wrappedTool.Tool(ctx)
}

func (t *TemporalToolProxy) NeedApproval() bool {
	return t.wrappedTool.NeedApproval()
}

func BuildTemporalToolsList(workflowCtx workflow.Context, config *agent_config.ToolConfig) []core.Tool {
	var toolList []core.Tool
	if config == nil {
		return toolList
	}

	// image, web search and code execution are LLM provider's tool so no need to temporalize them
	if config.ImageGeneration != nil && config.ImageGeneration.Enabled {
		toolList = append(toolList, tools.NewImageGenerationTool())
	}

	if config.WebSearch != nil && config.WebSearch.Enabled {
		toolList = append(toolList, tools.NewWebSearchTool())
	}

	if config.CodeExecution != nil && config.CodeExecution.Enabled {
		toolList = append(toolList, tools.NewCodeExecutionTool())
	}

	if config.Sandbox != nil && config.Sandbox.Enabled {
		image := os.Getenv("SANDBOX_DEFAULT_IMAGE")
		if config.Sandbox.DockerImage != nil {
			image = *config.Sandbox.DockerImage
		}

		toolList = append(toolList, NewTemporalSandboxToolProxy(workflowCtx, image))
	}

	return toolList
}
