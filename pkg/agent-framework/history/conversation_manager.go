package history

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services/conversation"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var convTracer = otel.Tracer("ConversationManager")

type ConversationPersistenceManager interface {
	LoadMessages(ctx context.Context, projectID uuid.UUID, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error)
	SaveMessages(ctx context.Context, projectID uuid.UUID, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error
	SaveSummary(ctx context.Context, projectID uuid.UUID, namespace string, summary conversation.Summary) error
}

type CommonConversationManager struct {
	ConversationPersistenceManager

	projectID      uuid.UUID
	namespace      string
	conversationId string
	msgId          string
	previousMsgId  string
	msgIdToRunId   map[string]string
	threadId       string

	convMessages []conversation.ConversationMessage
	oldMessages  []responses.InputMessageUnion
	newMessages  []responses.InputMessageUnion
	usage        *responses.Usage

	summarizer core.HistorySummarizer
	summaries  *core.SummaryResult
}

func NewConversationManager(p ConversationPersistenceManager, projectID uuid.UUID, namespace, msgId, previousMsgId string, opts ...ConversationManagerOptions) *CommonConversationManager {
	cm := &CommonConversationManager{
		ConversationPersistenceManager: p,
		projectID:                      projectID,
		namespace:                      namespace,
		msgId:                          msgId,
		previousMsgId:                  previousMsgId,
		msgIdToRunId:                   make(map[string]string),
	}

	for _, o := range opts {
		o(cm)
	}

	return cm
}

type ConversationManagerOptions func(*CommonConversationManager)

func WithConversationID(conversationId string) ConversationManagerOptions {
	return func(cm *CommonConversationManager) {
		cm.conversationId = conversationId
	}
}

func WithSummarizer(summarizer core.HistorySummarizer) ConversationManagerOptions {
	return func(cm *CommonConversationManager) {
		cm.summarizer = summarizer
	}
}

func (cm *CommonConversationManager) AddMessages(ctx context.Context, messages []responses.InputMessageUnion, usage *responses.Usage) {
	cm.newMessages = append(cm.newMessages, messages...)

	if usage != nil {
		cm.usage = usage
	}
}

func (cm *CommonConversationManager) GetMessages(ctx context.Context) ([]responses.InputMessageUnion, error) {
	// Process messages with summarizer if available
	if cm.summarizer != nil {
		summaryResult, err := cm.summarizer.Summarize(ctx, cm.msgIdToRunId, cm.oldMessages, cm.usage)
		if err != nil {
			return nil, err
		}

		// If a summary was created, track it for saving later and apply it to messages
		if summaryResult != nil {
			cm.summaries = summaryResult
			if summaryResult.Summary == nil {
				cm.oldMessages = summaryResult.MessagesToKeep
			} else {
				cm.oldMessages = append([]responses.InputMessageUnion{*summaryResult.Summary}, summaryResult.MessagesToKeep...)
			}
		}
	}

	return append(cm.oldMessages, cm.newMessages...), nil
}

func (cm *CommonConversationManager) LoadMessages(ctx context.Context) ([]responses.InputMessageUnion, error) {
	ctx, span := convTracer.Start(ctx, "ConversationManager.DB.LoadMessages")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", cm.namespace),
		attribute.String("previous_msg_id", cm.previousMsgId),
		attribute.String("project_id", cm.projectID.String()),
	)

	if cm.ConversationPersistenceManager == nil {
		span.SetAttributes(attribute.Bool("persistence_nil", true))
		return []responses.InputMessageUnion{}, nil
	}

	// Don't have to reload
	if len(cm.oldMessages) > 0 {
		return cm.oldMessages, nil
	}

	convMessages, err := cm.ConversationPersistenceManager.LoadMessages(ctx, cm.projectID, cm.namespace, cm.previousMsgId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("conversation_messages_count", len(convMessages)))

	messages := []responses.InputMessageUnion{}
	var usage *responses.Usage
	for _, msg := range convMessages {
		for _, m := range msg.Messages {
			cm.msgIdToRunId[m.ID()] = msg.MessageID
		}
		cm.threadId = msg.ThreadID

		messages = append(messages, msg.Messages...)
		if usageData, ok := msg.Meta["usage"].(map[string]any); ok {
			b, err := sonic.Marshal(usageData)
			if err != nil {
				continue
			}

			sonic.Unmarshal(b, &usage)
		}
	}

	cm.convMessages = convMessages
	cm.oldMessages = messages
	cm.usage = usage

	return messages, nil
}

func (cm *CommonConversationManager) SaveMessages(ctx context.Context, meta map[string]any) error {
	ctx, span := convTracer.Start(ctx, "ConversationManager.DB.SaveMessages")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", cm.namespace),
		attribute.String("msg_id", cm.msgId),
		attribute.Int("new_messages_count", len(cm.newMessages)),
		attribute.Bool("has_summary", cm.summaries != nil),
	)

	if cm.ConversationPersistenceManager == nil {
		span.SetAttributes(attribute.Bool("persistence_nil", true))
		return nil
	}

	if cm.summaries != nil {
		sum := conversation.Summary{
			ID:                      cm.summaries.SummaryID,
			ThreadID:                cm.threadId,
			LastSummarizedMessageID: cm.summaries.LastSummarizedMessageID,
			CreatedAt:               time.Now(),
			Meta: map[string]any{
				"is_summary": true,
			},
		}

		if cm.summaries.Summary != nil {
			sum.SummaryMessage = *cm.summaries.Summary
		}

		err := cm.ConversationPersistenceManager.SaveSummary(ctx, cm.projectID, cm.namespace, sum)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		cm.summaries = nil
	}

	err := cm.ConversationPersistenceManager.SaveMessages(ctx, cm.projectID, cm.namespace, cm.msgId, cm.previousMsgId, cm.conversationId, cm.newMessages, meta)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	cm.msgId = uuid.NewString()
	cm.oldMessages = append(cm.oldMessages, cm.newMessages...)
	cm.newMessages = []responses.InputMessageUnion{}

	return nil
}
