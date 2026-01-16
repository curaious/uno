package temporal_agent_builder

import (
	"context"

	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (b *AgentBuilder) Summarize(ctx context.Context, projectID uuid.UUID, config *agent_config.HistoryConfig, msgIdToRunId map[string]string, messages []responses.InputMessageUnion, usage *responses.Usage) (*core.SummaryResult, error) {
	conversationManager, err := builder.BuildConversationManager(b.svc, projectID, b.llmGateway, config)
	if err != nil {
		return nil, err
	}

	return conversationManager.Summarizer.Summarize(ctx, msgIdToRunId, messages, usage)
}

type TemporalConversationSummarizerProxy struct {
	workflowCtx workflow.Context
	projectID   uuid.UUID
	config      *agent_config.HistoryConfig
}

func NewTemporalConversationSummarizerProxy(workflowCtx workflow.Context, projectID uuid.UUID, config *agent_config.HistoryConfig) core.HistorySummarizer {
	return &TemporalConversationSummarizerProxy{
		workflowCtx: workflowCtx,
		projectID:   projectID,
		config:      config,
	}
}

func (t *TemporalConversationSummarizerProxy) Summarize(ctx context.Context, msgIdToRunId map[string]string, messages []responses.InputMessageUnion, usage *responses.Usage) (*core.SummaryResult, error) {
	var summaryResult *core.SummaryResult
	err := workflow.ExecuteActivity(t.workflowCtx, "Summarize", t.projectID, t.config, msgIdToRunId, messages, usage).Get(t.workflowCtx, &summaryResult)
	if err != nil {
		return nil, err
	}

	return summaryResult, nil
}
