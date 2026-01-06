package core

import (
	"context"

	"github.com/curaious/uno/pkg/llm/responses"
)

type Tool interface {
	Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error)
	Tool(ctx context.Context) *responses.ToolUnion
	NeedApproval() bool
}

type BaseTool struct {
	*responses.ToolUnion
	RequiresApproval bool
}

func (t *BaseTool) NeedApproval() bool {
	return t.RequiresApproval
}
