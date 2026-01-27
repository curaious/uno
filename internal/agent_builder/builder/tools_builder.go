package builder

import (
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/tools"
	"github.com/curaious/uno/pkg/sandbox"
)

func BuildToolsList(config *agent_config.ToolConfig, svc sandbox.Manager) []core.Tool {
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

	if config.CodeExecution != nil && config.CodeExecution.Enabled {
		toolList = append(toolList, tools.NewCodeExecutionTool())
	}

	if config.Sandbox != nil && config.Sandbox.Enabled {
		image := "uno-sandbox:v6"
		if config.Sandbox.DockerImage != nil {
			image = *config.Sandbox.DockerImage
		}

		toolList = append(toolList, tools.NewSandboxTool(svc, image))
	}

	return toolList
}
