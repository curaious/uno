package temporal_agent_builder

import (
	"context"
	"fmt"

	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/workflow"
)

func (b *AgentBuilder) MCPListTools(ctx context.Context, config *agent_config.MCPServerConfig, runContext map[string]any) ([]core.BaseTool, error) {
	client, err := builder.BuildMCPClient(config)
	if err != nil {
		return nil, err
	}

	mcpTools, err := client.ListTools(ctx, runContext)
	if err != nil {
		return nil, err
	}

	var tools []core.BaseTool
	for _, tool := range mcpTools {
		tu := tool.Tool(ctx)
		tools = append(tools, core.BaseTool{
			ToolUnion:        *tu,
			RequiresApproval: tool.NeedApproval(),
		})
	}

	return tools, nil
}

func (b *AgentBuilder) MCPCallTool(ctx context.Context, config *agent_config.MCPServerConfig, params *core.ToolCall, runContext map[string]any) (*responses.FunctionCallOutputMessage, error) {
	client, err := builder.BuildMCPClient(config)
	if err != nil {
		return nil, err
	}

	mcpTools, err := client.ListTools(ctx, runContext)
	if err != nil {
		return nil, err
	}

	for _, tool := range mcpTools {
		if t := tool.Tool(ctx); t != nil && t.OfFunction != nil && params.Name == t.OfFunction.Name {
			return tool.Execute(ctx, params)
		}
	}

	return nil, fmt.Errorf("no tool found with name %s", params.Name)
}

type TemporalMCPProxy struct {
	workflowCtx workflow.Context
	config      *agent_config.MCPServerConfig
}

func (t *TemporalMCPProxy) GetName() string {
	return t.config.Name
}

func NewTemporalMCPProxy(workflowCtx workflow.Context, config *agent_config.MCPServerConfig) *TemporalMCPProxy {
	return &TemporalMCPProxy{
		workflowCtx: workflowCtx,
		config:      config,
	}
}

func (t *TemporalMCPProxy) ListTools(ctx context.Context, runContext map[string]any) ([]core.Tool, error) {
	toolDefs := []core.BaseTool{}
	err := workflow.ExecuteActivity(t.workflowCtx, "MCPListTools", t.config, runContext).Get(t.workflowCtx, &toolDefs)
	if err != nil {
		return nil, err
	}

	var toolList []core.Tool
	for _, toolDef := range toolDefs {
		toolList = append(toolList, NewTemporalMCPToolProxy(t.workflowCtx, t.config, runContext, toolDef))
	}

	return toolList, nil
}

type TemporalMCPToolProxy struct {
	workflowCtx workflow.Context
	config      *agent_config.MCPServerConfig
	runContext  map[string]any
	*core.BaseTool
}

func NewTemporalMCPToolProxy(workflowCtx workflow.Context, config *agent_config.MCPServerConfig, runContext map[string]any, baseTool core.BaseTool) *TemporalMCPToolProxy {
	return &TemporalMCPToolProxy{
		workflowCtx: workflowCtx,
		config:      config,
		runContext:  runContext,
		BaseTool:    &baseTool,
	}
}

func (t *TemporalMCPToolProxy) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	var output *responses.FunctionCallOutputMessage
	err := workflow.ExecuteActivity(t.workflowCtx, "MCPCallTool", t.config, params, t.runContext).Get(t.workflowCtx, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}
