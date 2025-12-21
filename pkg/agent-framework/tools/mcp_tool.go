package tools

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	tracer = otel.Tracer("MCPTool")
)

type MCPServer struct {
	Client *client.Client `json:"-"`
	Tools  []mcp.Tool     `json:"-"`
	Meta   *mcp.Meta      `json:"-"`
}

func NewInProcessMCPServer(ctx context.Context, client *client.Client, headers map[string]any) (*MCPServer, error) {
	err := client.Start(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.Initialize(ctx, mcp.InitializeRequest{
		Request: mcp.Request{},
		Params:  mcp.InitializeParams{},
	})
	if err != nil {
		return nil, err
	}

	tools, err := client.ListTools(ctx, mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{},
	})
	if err != nil {
		return nil, err
	}

	return &MCPServer{
		Tools:  tools.Tools,
		Client: client,
		Meta: &mcp.Meta{
			AdditionalFields: headers,
		},
	}, nil
}

func NewMCPServer(ctx context.Context, endpoint string, headers map[string]string) (*MCPServer, error) {
	client, err := client.NewSSEMCPClient(
		endpoint,
		client.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}

	err = client.Start(ctx)
	if err != nil {
		return nil, err
	}

	h := http.Header{}
	for k, v := range headers {
		h.Add(k, v)
	}

	_, err = client.Initialize(ctx, mcp.InitializeRequest{
		Request: mcp.Request{},
		Params:  mcp.InitializeParams{},
		Header:  h,
	})
	if err != nil {
		return nil, err
	}

	tools, err := client.ListTools(ctx, mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{},
		Header:           h,
	})
	if err != nil {
		return nil, err
	}

	return &MCPServer{
		Tools:  tools.Tools,
		Client: client,
	}, nil
}

func (srv *MCPServer) GetTools(toolFilter ...string) []core.Tool {
	mcpTools := []core.Tool{}

	for _, tool := range srv.Tools {
		if len(toolFilter) > 0 && !slices.Contains(toolFilter, tool.Name) {
			continue
		}
		mcpTools = append(mcpTools, NewMcpTool(tool, srv.Client, srv.Meta))
	}

	return mcpTools
}

type McpTool struct {
	*core.BaseTool
	Client *client.Client `json:"-"`
	Meta   *mcp.Meta      `json:"-"`
}

func NewMcpTool(t mcp.Tool, cli *client.Client, Meta *mcp.Meta) *McpTool {
	inputSchema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	inputSchemaBytes, err := sonic.Marshal(t.InputSchema)
	if err == nil {
		_ = sonic.Unmarshal(inputSchemaBytes, &inputSchema)
	}

	outputSchema := map[string]any{}
	outputSchemaBytes, err := t.RawOutputSchema.MarshalJSON()
	if err == nil {
		_ = sonic.Unmarshal(outputSchemaBytes, &outputSchema)
	}

	return &McpTool{
		BaseTool: &core.BaseTool{
			ToolUnion: &responses.ToolUnion{
				OfFunction: &responses.FunctionTool{
					Name:        t.Name,
					Description: utils.Ptr(t.Description),
					Parameters:  inputSchema,
					Strict:      utils.Ptr(false),
				},
			},
		},
		Client: cli,
		Meta:   Meta,
	}
}

func (c *McpTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	ctx, span := tracer.Start(ctx, "McpTool: "+params.Name)
	defer span.End()

	span.SetAttributes(attribute.String("input", params.Arguments))

	var args map[string]any
	if params.Arguments != "" {
		err := sonic.Unmarshal([]byte(params.Arguments), &args)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.String("output", err.Error()))
			return &responses.FunctionCallOutputMessage{
				ID:     params.ID,
				CallID: params.CallID,
				Output: responses.FunctionCallOutputContentUnion{
					OfString: utils.Ptr(err.Error()),
				},
			}, nil
		}
	}

	// Call the MCP tool
	res, err := c.Client.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{},
		Params: mcp.CallToolParams{
			Name:      params.Name,
			Arguments: args,
			Meta:      c.Meta,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("output", err.Error()))
		return &responses.FunctionCallOutputMessage{
			ID:     params.ID,
			CallID: params.CallID,
			Output: responses.FunctionCallOutputContentUnion{
				OfString: utils.Ptr(err.Error()),
			},
		}, nil
	}

	// Return the tool result
	for _, r := range res.Content {
		switch r.(type) {
		case mcp.TextContent:
			out := &responses.FunctionCallOutputMessage{
				ID:     params.ID,
				CallID: params.CallID,
				Output: responses.FunctionCallOutputContentUnion{
					OfString: utils.Ptr(r.(mcp.TextContent).Text),
				},
			}
			outStr, _ := sonic.Marshal(out)
			span.SetAttributes(attribute.String("output", string(outStr)))
			return out, nil
		}
	}

	err = errors.New("missing mcp tool result")
	span.RecordError(err)
	return nil, err
}
