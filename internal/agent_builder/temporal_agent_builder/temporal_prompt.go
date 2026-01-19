package temporal_agent_builder

import (
	"context"

	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (b *AgentBuilder) GetPrompt(ctx context.Context, projectID uuid.UUID, config *agent_config.PromptConfig, runContext map[string]any) (string, error) {
	instruction := builder.BuildPrompt(b.svc.Prompt, projectID, config)
	return instruction.GetPrompt(ctx, runContext)
}

type TemporalPromptProxy struct {
	workflowCtx workflow.Context
	config      *agent_config.PromptConfig
	projectID   uuid.UUID
}

func NewTemporalPromptProxy(workflowCtx workflow.Context, projectID uuid.UUID, config *agent_config.PromptConfig) *TemporalPromptProxy {
	return &TemporalPromptProxy{
		workflowCtx: workflowCtx,
		config:      config,
		projectID:   projectID,
	}
}

func (p *TemporalPromptProxy) GetPrompt(ctx context.Context, data map[string]any) (string, error) {
	var promptString string
	err := workflow.ExecuteActivity(p.workflowCtx, "GetPrompt", p.projectID, p.config, data).Get(p.workflowCtx, &promptString)
	return promptString, err
}
