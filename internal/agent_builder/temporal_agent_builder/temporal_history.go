package temporal_agent_builder

import (
	"context"

	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (b *AgentBuilder) LoadMessages(ctx context.Context, projectID uuid.UUID, config *agent_config.HistoryConfig, namespace string, previousMessageId string) ([]conversation.ConversationMessage, error) {
	conversationManager, err := builder.BuildConversationManager(b.svc, projectID, b.llmGateway, config, "")
	if err != nil {
		return nil, err
	}

	return conversationManager.ConversationPersistenceAdapter.LoadMessages(ctx, namespace, previousMessageId)
}

func (b *AgentBuilder) SaveMessages(ctx context.Context, projectID uuid.UUID, config *agent_config.HistoryConfig, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	conversationManager, err := builder.BuildConversationManager(b.svc, projectID, b.llmGateway, config, "")
	if err != nil {
		return err
	}

	return conversationManager.ConversationPersistenceAdapter.SaveMessages(ctx, namespace, msgId, previousMsgId, conversationId, messages, meta)
}

func (b *AgentBuilder) SaveSummary(ctx context.Context, projectID uuid.UUID, config *agent_config.HistoryConfig, namespace string, summary conversation.Summary) error {
	conversationManager, err := builder.BuildConversationManager(b.svc, projectID, b.llmGateway, config, "")
	if err != nil {
		return err
	}

	return conversationManager.ConversationPersistenceAdapter.SaveSummary(ctx, namespace, summary)
}

type TemporalConversationPersistenceProxy struct {
	workflowCtx workflow.Context
	projectID   uuid.UUID
	config      *agent_config.HistoryConfig
}

func NewTemporalConversationPersistenceProxy(workflowCtx workflow.Context, projectID uuid.UUID, config *agent_config.HistoryConfig) *TemporalConversationPersistenceProxy {
	return &TemporalConversationPersistenceProxy{
		workflowCtx: workflowCtx,
		projectID:   projectID,
		config:      config,
	}
}

func (t *TemporalConversationPersistenceProxy) NewRunID(ctx context.Context) string {
	idAny := workflow.SideEffect(t.workflowCtx, func(ctx workflow.Context) interface{} {
		return uuid.NewString()
	})

	var id string
	if err := idAny.Get(&idAny); err != nil {
		return uuid.NewString() // ideally, we won't get here as uuid.NewString() is not supposed to throw errors
	}

	return id
}

func (t *TemporalConversationPersistenceProxy) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	var messages []conversation.ConversationMessage
	err := workflow.ExecuteActivity(t.workflowCtx, "LoadMessages", t.projectID, t.config, namespace, previousMessageID).Get(t.workflowCtx, &messages)
	if err != nil {
		return messages, err
	}

	return messages, nil
}

func (t *TemporalConversationPersistenceProxy) SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return workflow.ExecuteActivity(t.workflowCtx, "SaveMessages", t.projectID, t.config, namespace, msgId, previousMsgId, conversationId, messages, meta).Get(t.workflowCtx, nil)
}

func (t *TemporalConversationPersistenceProxy) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return workflow.ExecuteActivity(t.workflowCtx, "SaveSummary", t.projectID, t.config, namespace, summary).Get(t.workflowCtx, nil)
}
