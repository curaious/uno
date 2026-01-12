package agents

import (
	"context"
	"fmt"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
	"github.com/curaious/uno/pkg/llm/responses"
)

func PrepareMCPTools(ctx context.Context, mcpServers []*mcpclient.MCPClient, runContext map[string]any) ([]core.Tool, error) {
	coreTools := []core.Tool{}
	if mcpServers != nil {
		for _, mcpServer := range mcpServers {
			cli, err := mcpServer.GetClient(ctx, runContext)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize MCP server: %w", err)
			}
			coreTools = append(coreTools, cli.GetTools()...)
		}
	}

	return coreTools, nil
}

func LoadMessages(ctx context.Context, convHistory *history.ConversationRunManager, namespace string, previousMessageID string) ([]responses.InputMessageUnion, error) {
	messages, err := convHistory.LoadMessages(ctx, namespace, previousMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	return messages, nil
}
