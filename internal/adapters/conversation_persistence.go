package adapters

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
)

type InternalConversationPersistence struct {
	svc       *conversation.ConversationService
	projectID uuid.UUID
}

func NewInternalConversationPersistence(svc *conversation.ConversationService, projectID uuid.UUID) *InternalConversationPersistence {
	return &InternalConversationPersistence{
		svc:       svc,
		projectID: projectID,
	}
}

// LoadMessages implements core.ChatHistory
func (p *InternalConversationPersistence) LoadMessages(ctx context.Context, namespace string, previousMessageId string) ([]conversation.ConversationMessage, error) {
	// If no previous message ID, return empty list
	if previousMessageId == "" {
		return []conversation.ConversationMessage{}, nil
	}

	return p.svc.GetAllMessagesTillRun(ctx, &conversation.GetMessagesRequest{
		ProjectID:         p.projectID,
		Namespace:         namespace,
		PreviousMessageID: previousMessageId,
	})
}

// SaveMessages implements core.ChatHistory
func (p *InternalConversationPersistence) SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return p.svc.AddMessages(ctx, &conversation.AddMessageRequest{
		ProjectID:         p.projectID,
		Namespace:         namespace,
		MessageID:         msgId,
		PreviousMessageID: previousMsgId,
		Messages:          messages,
		Meta:              meta,
		ConversationID:    conversationId,
	})
}

// SaveSummary
func (p *InternalConversationPersistence) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return p.svc.CreateSummary(ctx, p.projectID, namespace, summary)
}
