package core

import (
	"context"

	"github.com/curaious/uno/pkg/llm/responses"
)

type ToolCall struct {
	*responses.FunctionCallMessage
	AgentName      string `json:"agent_name"`
	AgentVersion   string `json:"agent_version"`
	Namespace      string `json:"namespace"`
	ConversationID string `json:"conversation_id"`
}

type Tool interface {
	Execute(ctx context.Context, params *ToolCall) (*responses.FunctionCallOutputMessage, error)
	Tool(ctx context.Context) *responses.ToolUnion
	NeedApproval() bool
}

type BaseTool struct {
	ToolUnion        responses.ToolUnion
	RequiresApproval bool
}

func (t *BaseTool) NeedApproval() bool {
	return t.RequiresApproval
}

func (t *BaseTool) Tool(ctx context.Context) *responses.ToolUnion {
	return &t.ToolUnion
}
