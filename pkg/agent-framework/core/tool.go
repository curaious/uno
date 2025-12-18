package core

import (
	"context"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type Tool interface {
	Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error)
	AsFunctionTool(ctx context.Context) *responses.ToolUnion
}

type BaseTool struct {
	Type         string
	Name         string
	Description  string
	InputSchema  map[string]any
	OutputSchema map[string]any
}

func (h *BaseTool) AsFunctionTool(ctx context.Context) *responses.ToolUnion {
	return &responses.ToolUnion{OfFunction: &responses.FunctionTool{
		Type:        "function",
		Name:        h.Name,
		Description: utils.Ptr(h.Description),
		Parameters:  h.InputSchema,
		Strict:      utils.Ptr(false),
	}}
}
