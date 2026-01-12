package history

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var convTracer = otel.Tracer("ConversationManager")

type ConversationPersistenceAdapter interface {
	LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error)
	SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error
	SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error
}

type CommonConversationManager struct {
	ConversationPersistenceAdapter ConversationPersistenceAdapter
	Options                        []ConversationManagerOptions
}

func NewConversationManager(p ConversationPersistenceAdapter, opts ...ConversationManagerOptions) *CommonConversationManager {
	return &CommonConversationManager{
		ConversationPersistenceAdapter: p,
		Options:                        opts,
	}
}

type ConversationManagerOptions func(*ConversationRunManager)

func WithConversationID(conversationId string) ConversationManagerOptions {
	return func(cm *ConversationRunManager) {
		cm.conversationId = conversationId
	}
}

func WithSummarizer(summarizer core.HistorySummarizer) ConversationManagerOptions {
	return func(cm *ConversationRunManager) {
		cm.summarizer = summarizer
	}
}

func WithPersistence(customAdapter ConversationPersistenceAdapter) ConversationManagerOptions {
	return func(cm *ConversationRunManager) {
		cm.ConversationPersistenceAdapter = customAdapter
	}
}

func WithMessageID(msgId string) ConversationManagerOptions {
	return func(cm *ConversationRunManager) {
		cm.msgId = msgId
	}
}

type ConversationRunManager struct {
	ConversationPersistenceAdapter

	namespace      string
	conversationId string
	msgId          string
	previousMsgId  string
	msgIdToRunId   map[string]string
	threadId       string

	convMessages    []conversation.ConversationMessage
	oldMessages     []responses.InputMessageUnion
	newMessages     []responses.InputMessageUnion
	usage           *responses.Usage
	lastMessageMeta map[string]any

	summarizer core.HistorySummarizer
	summaries  *core.SummaryResult
}

func NewRun(persistence ConversationPersistenceAdapter, opts ...ConversationManagerOptions) *ConversationRunManager {
	cr := &ConversationRunManager{
		ConversationPersistenceAdapter: persistence,
		msgIdToRunId:                   make(map[string]string),
	}

	for _, o := range opts {
		o(cr)
	}

	return cr
}

func (cm *ConversationRunManager) AddMessages(ctx context.Context, messages []responses.InputMessageUnion, usage *responses.Usage) {
	cm.newMessages = append(cm.newMessages, messages...)

	if usage != nil {
		cm.usage = usage
	}
}

func (cm *ConversationRunManager) GetMessages(ctx context.Context) ([]responses.InputMessageUnion, error) {
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

func (cm *ConversationRunManager) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]responses.InputMessageUnion, error) {
	ctx, span := convTracer.Start(ctx, "ConversationManager.DB.LoadMessages")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", namespace),
		attribute.String("previous_msg_id", previousMessageID),
	)

	if cm.ConversationPersistenceAdapter == nil {
		span.SetAttributes(attribute.Bool("persistence_nil", true))
		return []responses.InputMessageUnion{}, nil
	}

	// Don't have to reload
	if len(cm.oldMessages) > 0 {
		return cm.oldMessages, nil
	}

	convMessages, err := cm.ConversationPersistenceAdapter.LoadMessages(ctx, namespace, previousMessageID)
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

		// Store the most recent message's meta for run state loading
		// The last message in the chain contains the current run state
		if msg.Meta != nil {
			cm.lastMessageMeta = msg.Meta
		}
	}

	// Initialize lastMessageMeta if no messages were found
	if cm.lastMessageMeta == nil {
		cm.lastMessageMeta = make(map[string]any)
	} else {
		runState := core.LoadRunStateFromMeta(cm.lastMessageMeta)
		if !runState.IsComplete() {
			cm.msgId = previousMessageID
		}
	}

	cm.namespace = namespace
	cm.previousMsgId = previousMessageID
	cm.convMessages = convMessages
	cm.oldMessages = messages
	cm.usage = usage

	return messages, nil
}

// GetMeta returns the meta from the most recent message
func (cm *ConversationRunManager) GetMeta() map[string]any {
	return cm.lastMessageMeta
}

// GetMessageID returns the current run id
func (cm *ConversationRunManager) GetMessageID() string {
	return cm.msgId
}

func (cm *ConversationRunManager) SaveMessages(ctx context.Context, meta map[string]any) error {
	ctx, span := convTracer.Start(ctx, "ConversationManager.DB.SaveMessages")
	defer span.End()

	span.SetAttributes(
		attribute.String("namespace", cm.namespace),
		attribute.String("msg_id", cm.msgId),
		attribute.Int("new_messages_count", len(cm.newMessages)),
		attribute.Bool("has_summary", cm.summaries != nil),
	)

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

		if cm.ConversationPersistenceAdapter != nil {
			err := cm.ConversationPersistenceAdapter.SaveSummary(ctx, cm.namespace, sum)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}

		cm.summaries = nil
	}

	if cm.ConversationPersistenceAdapter != nil {
		err := cm.ConversationPersistenceAdapter.SaveMessages(ctx, cm.namespace, cm.msgId, cm.previousMsgId, cm.conversationId, cm.newMessages, meta)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	runState := core.LoadRunStateFromMeta(meta)
	if runState.IsComplete() {
		cm.previousMsgId = cm.msgId
		cm.msgId = uuid.NewString()
	}

	cm.lastMessageMeta = meta
	cm.oldMessages = append(cm.oldMessages, cm.newMessages...)
	cm.newMessages = []responses.InputMessageUnion{}

	return nil
}
