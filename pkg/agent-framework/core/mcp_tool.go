package core

import (
	"context"

	"github.com/curaious/uno/pkg/llm/responses"
)

type MCPToolset interface {
	ListTools(ctx context.Context) ([]responses.ToolUnion, error)
}
