package adapters

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	tracer = otel.Tracer("InternalAdapters")
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

// NewConversationID generates a unique ID for a conversation
func (p *InternalConversationPersistence) NewConversationID(ctx context.Context) string {
	return uuid.NewString()
}

// NewRunID generates a unique ID for a run
func (p *InternalConversationPersistence) NewRunID(ctx context.Context) string {
	return uuid.NewString()
}

// LoadMessages implements core.ChatHistory
func (p *InternalConversationPersistence) LoadMessages(ctx context.Context, namespace string, previousMessageId string) ([]conversation.ConversationMessage, error) {
	ctx, span := tracer.Start(ctx, "InternalConversationPersistence.LoadMessages")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", namespace),
		attribute.String("previous_message_id", previousMessageId),
	)

	// If no previous message ID, return empty list
	if previousMessageId == "" {
		return []conversation.ConversationMessage{}, nil
	}

	messages, err := p.svc.GetAllMessagesTillRun(ctx, &conversation.GetMessagesRequest{
		ProjectID:         p.projectID,
		Namespace:         namespace,
		PreviousMessageID: previousMessageId,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("conversation_messages_count", len(messages)))

	return messages, nil
}

// SaveMessages implements core.ChatHistory
func (p *InternalConversationPersistence) SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	ctx, span := tracer.Start(ctx, "InternalConversationPersistence.SaveMessages")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", namespace),
		attribute.String("previous_message_id", previousMsgId),
		attribute.String("conversation_id", conversationId),
		attribute.Int("messages_count", len(messages)),
	)

	err := p.svc.AddMessages(ctx, &conversation.AddMessageRequest{
		ProjectID:         p.projectID,
		Namespace:         namespace,
		MessageID:         msgId,
		PreviousMessageID: previousMsgId,
		Messages:          messages,
		Meta:              meta,
		ConversationID:    conversationId,
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

// SaveSummary
func (p *InternalConversationPersistence) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	ctx, span := tracer.Start(ctx, "InternalConversationPersistence.SaveSummary")
	defer span.End()

	err := p.svc.CreateSummary(ctx, p.projectID, namespace, summary)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
