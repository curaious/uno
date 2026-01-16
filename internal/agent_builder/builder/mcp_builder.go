package builder

import (
	"context"

	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
)

func BuildMCPClient(config *agent_config.MCPServerConfig) (*mcpclient.MCPClient, error) {
	options := []mcpclient.McpServerOption{}
	if config.Headers != nil && len(config.Headers) > 0 {
		options = append(options, mcpclient.WithHeaders(config.Headers))
	}

	if config.ToolFilters != nil && len(config.ToolFilters) > 0 {
		options = append(options, mcpclient.WithToolFilter(config.ToolFilters...))
	}

	if config.ToolsRequiringHumanApproval != nil && len(config.ToolsRequiringHumanApproval) > 0 {
		options = append(options, mcpclient.WithApprovalRequiredTools(config.ToolsRequiringHumanApproval...))
	}

	mcpServer, err := mcpclient.NewSSEClient(context.Background(), config.Endpoint, options...)
	if err != nil {
		return nil, err
	}

	return mcpServer, nil
}
