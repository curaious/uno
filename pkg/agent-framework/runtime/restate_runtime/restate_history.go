package restate_runtime

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
)

type RestateHistory struct {
	restateCtx         restate.WorkflowContext
	wrappedPersistence history.ConversationPersistenceAdapter
}

func NewRestateConversationPersistence(restateCtx restate.WorkflowContext, wrappedPersistence history.ConversationPersistenceAdapter) *RestateHistory {
	return &RestateHistory{
		restateCtx:         restateCtx,
		wrappedPersistence: wrappedPersistence,
	}
}

func (t *RestateHistory) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	return restate.Run(t.restateCtx, func(ctx restate.RunContext) ([]conversation.ConversationMessage, error) {
		return t.wrappedPersistence.LoadMessages(ctx, namespace, previousMessageID)
	}, restate.WithName("LoadMessages"))
}

func (t *RestateHistory) SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	_, err := restate.Run(t.restateCtx, func(ctx restate.RunContext) (any, error) {
		return nil, t.wrappedPersistence.SaveMessages(ctx, namespace, msgId, previousMsgId, conversationId, messages, meta)
	}, restate.WithName("SaveMessages"))
	return err
}

func (t *RestateHistory) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	_, err := restate.Run(t.restateCtx, func(ctx restate.RunContext) (any, error) {
		return nil, t.wrappedPersistence.SaveSummary(ctx, namespace, summary)
	}, restate.WithName("SaveSummary"))
	return err
}
