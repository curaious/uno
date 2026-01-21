package tools

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
)

type WebSearchTool struct {
	*core.BaseTool
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{}
}

func (t *WebSearchTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return nil, nil
}

func (t *WebSearchTool) Tool(ctx context.Context) *responses.ToolUnion {
	return &responses.ToolUnion{OfWebSearch: &responses.WebSearchTool{}}
}
