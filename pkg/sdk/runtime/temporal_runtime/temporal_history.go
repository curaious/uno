package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type TemporalHistory struct {
	wrappedPersistence history.ConversationPersistenceAdapter
}

func NewTemporalConversationPersistence(wrappedPersistence history.ConversationPersistenceAdapter) *TemporalHistory {
	return &TemporalHistory{
		wrappedPersistence: wrappedPersistence,
	}
}

func (t *TemporalHistory) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	return t.wrappedPersistence.LoadMessages(ctx, namespace, previousMessageID)
}

func (t *TemporalHistory) SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return t.wrappedPersistence.SaveMessages(ctx, namespace, msgId, previousMsgId, conversationId, messages, meta)
}

func (t *TemporalHistory) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return t.wrappedPersistence.SaveSummary(ctx, namespace, summary)
}

type TemporalConversationPersistenceProxy struct {
	workflowCtx workflow.Context
	prefix      string
}

func NewTemporalConversationPersistenceProxy(workflowCtx workflow.Context, prefix string) history.ConversationPersistenceAdapter {
	return &TemporalConversationPersistenceProxy{
		workflowCtx: workflowCtx,
		prefix:      prefix,
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
	err := workflow.ExecuteActivity(t.workflowCtx, t.prefix+"_LoadMessagesActivity", namespace, previousMessageID).Get(t.workflowCtx, &messages)
	if err != nil {
		return messages, err
	}

	return messages, nil
}

func (t *TemporalConversationPersistenceProxy) SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return workflow.ExecuteActivity(t.workflowCtx, t.prefix+"_SaveMessagesActivity", namespace, msgId, previousMsgId, conversationId, messages, meta).Get(t.workflowCtx, nil)
}

func (t *TemporalConversationPersistenceProxy) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return workflow.ExecuteActivity(t.workflowCtx, t.prefix+"_SaveSummaryActivity", namespace, summary).Get(t.workflowCtx, nil)
}
