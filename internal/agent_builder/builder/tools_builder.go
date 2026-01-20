package builder

import (
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/tools"
)

func BuildToolsList(config *agent_config.ToolConfig) []core.Tool {
	var toolList []core.Tool
	if config == nil {
		return toolList
	}

	if config.ImageGeneration != nil && config.ImageGeneration.Enabled {
		toolList = append(toolList, tools.NewImageGenerationTool())
	}

	if config.WebSearch != nil && config.WebSearch.Enabled {
		toolList = append(toolList, tools.NewWebSearchTool())
	}

	return toolList
}
