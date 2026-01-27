package restate_runtime

import (
	"context"
	"fmt"

	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
)

type RestateMCPServer struct {
	restateCtx       restate.WorkflowContext
	wrappedMcpServer agents.MCPToolset
}

func NewRestateMCPServer(restateCtx restate.WorkflowContext, wrappedMcpServer agents.MCPToolset) *RestateMCPServer {
	return &RestateMCPServer{
		restateCtx:       restateCtx,
		wrappedMcpServer: wrappedMcpServer,
	}
}

func (t *RestateMCPServer) GetName() string {
	return ""
}

func (t *RestateMCPServer) ListTools(ctx context.Context, runContext map[string]any) ([]core.Tool, error) {
	// TODO: `RestateMCPServer` is created per workflow, so we can connect to MCP and keep the connection
	toolDefs, err := restate.Run(t.restateCtx, func(ctx restate.RunContext) ([]core.BaseTool, error) {
		mcpTools, err := t.wrappedMcpServer.ListTools(ctx, runContext)
		if err != nil {
			return nil, err
		}

		var tools []core.BaseTool
		for _, tool := range mcpTools {
			tools = append(tools, core.BaseTool{
				ToolUnion:        *tool.Tool(ctx),
				RequiresApproval: tool.NeedApproval(),
			})
		}

		return tools, nil
	}, restate.WithName("MCPListTools"))
	if err != nil {
		return nil, err
	}

	var tools []core.Tool
	for _, tool := range toolDefs {
		tools = append(tools, NewRestateMCPTool(t.restateCtx, t.wrappedMcpServer, runContext, tool))
	}

	return tools, nil
}

type RestateMCPTool struct {
	restateCtx       restate.WorkflowContext
	runContext       map[string]any
	wrappedMcpServer agents.MCPToolset
	*core.BaseTool
}

func NewRestateMCPTool(restateCtx restate.WorkflowContext, wrappedMcpServer agents.MCPToolset, runContext map[string]any, baseTool core.BaseTool) *RestateMCPTool {
	return &RestateMCPTool{
		restateCtx:       restateCtx,
		runContext:       runContext,
		wrappedMcpServer: wrappedMcpServer,
		BaseTool:         &baseTool,
	}
}

func (t *RestateMCPTool) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	return restate.Run(t.restateCtx, func(ctx restate.RunContext) (*responses.FunctionCallOutputMessage, error) {
		mcpTools, err := t.wrappedMcpServer.ListTools(ctx, t.runContext)
		if err != nil {
			return nil, err
		}

		for _, tool := range mcpTools {
			if t := tool.Tool(ctx); t != nil && t.OfFunction != nil && params.Name == t.OfFunction.Name {
				return tool.Execute(ctx, params)
			}
		}

		return nil, fmt.Errorf("no restate tool found with name %s", params.Name)
	}, restate.WithName("MCPToolCall"))
}
