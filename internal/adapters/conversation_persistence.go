package adapters

import (
	"context"

	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services/conversation"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type InternalConversationPersistence struct {
	svc *conversation.ConversationService
}

func NewInternalConversationPersistence(svc *conversation.ConversationService) *InternalConversationPersistence {
	return &InternalConversationPersistence{
		svc: svc,
	}
}

// LoadMessages implements core.ChatHistory
func (p *InternalConversationPersistence) LoadMessages(ctx context.Context, projectID uuid.UUID, namespace string, previousMessageId string) ([]conversation.ConversationMessage, error) {
	// If no previous message ID, return empty list
	if previousMessageId == "" {
		return []conversation.ConversationMessage{}, nil
	}

	return p.svc.GetAllMessagesTillRun(ctx, &conversation.GetMessagesRequest{
		ProjectID:         projectID,
		Namespace:         namespace,
		PreviousMessageID: previousMessageId,
	})
}

// SaveMessages implements core.ChatHistory
func (p *InternalConversationPersistence) SaveMessages(ctx context.Context, projectID uuid.UUID, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return p.svc.AddMessages(ctx, &conversation.AddMessageRequest{
		ProjectID:         projectID,
		Namespace:         namespace,
		MessageID:         msgId,
		PreviousMessageID: previousMsgId,
		Messages:          messages,
		Meta:              meta,
		ConversationID:    conversationId,
	})
}

// SaveSummary
func (p *InternalConversationPersistence) SaveSummary(ctx context.Context, projectID uuid.UUID, namespace string, summary conversation.Summary) error {
	return p.svc.CreateSummary(ctx, projectID, namespace, summary)
}
