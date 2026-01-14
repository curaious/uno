package history

import (
	"context"
	"errors"
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
	NewRunID(ctx context.Context) string
	LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error)
	SaveMessages(ctx context.Context, namespace, msgId, previousMsgId, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error
	SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error
}

type CommonConversationManager struct {
	ConversationPersistenceAdapter ConversationPersistenceAdapter
	Summarizer                     core.HistorySummarizer

	Options []ConversationManagerOptions
}

func NewConversationManager(p ConversationPersistenceAdapter, opts ...ConversationManagerOptions) *CommonConversationManager {
	cm := &CommonConversationManager{
		ConversationPersistenceAdapter: p,
	}

	for _, o := range opts {
		o(cm)
	}

	return cm
}

type ConversationManagerOptions func(*CommonConversationManager)

func WithSummarizer(summarizer core.HistorySummarizer) ConversationManagerOptions {
	return func(cm *CommonConversationManager) {
		cm.Summarizer = summarizer
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
	RunState        *core.RunState

	summarizer core.HistorySummarizer
	summaries  *core.SummaryResult
}

func NewRun(ctx context.Context, cm *CommonConversationManager, namespace string, previousRunID string, messages []responses.InputMessageUnion) (*ConversationRunManager, error) {
	cr := &ConversationRunManager{
		ConversationPersistenceAdapter: cm.ConversationPersistenceAdapter,
		summarizer:                     cm.Summarizer,
		msgIdToRunId:                   make(map[string]string),
	}

	// Load messages
	_, err := cr.LoadMessages(ctx, namespace, previousRunID)
	if err != nil {
		return nil, err
	}

	// Load the run state
	var runID string
	if cr.RunState == nil || cr.RunState.IsComplete() {
		// Create a new run id
		runID = cr.ConversationPersistenceAdapter.NewRunID(ctx)
		cr.RunState = core.NewRunState()
		cr.AddMessages(ctx, messages, nil)
	} else {
		// Continuing the previous run
		runID = previousRunID

		if cr.RunState.CurrentStep == core.StepAwaitApproval {
			// Expect approval
			if len(messages) == 0 || messages[0].OfFunctionCallApprovalResponse == nil {
				return nil, errors.New("expected approval response message to resume the run")
			}

			// Transition to tool execution, as we have approval message
			cr.RunState.CurrentStep = core.StepExecuteTools
		}
	}

	// Store the run id
	cr.msgId = runID

	return cr, nil
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
	}

	cm.namespace = namespace
	cm.previousMsgId = previousMessageID
	cm.convMessages = convMessages
	cm.oldMessages = messages
	cm.usage = usage
	cm.RunState = core.LoadRunStateFromMeta(cm.lastMessageMeta)

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

func (cm *ConversationRunManager) TrackUsage(usage *responses.Usage) {
	cm.RunState.Usage.InputTokens += usage.InputTokens
	cm.RunState.Usage.OutputTokens += usage.OutputTokens
	cm.RunState.Usage.InputTokensDetails.CachedTokens += usage.InputTokensDetails.CachedTokens
	cm.RunState.Usage.TotalTokens += usage.TotalTokens
}
