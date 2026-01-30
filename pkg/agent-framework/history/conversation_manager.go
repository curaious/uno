package history

import (
	"context"
	"errors"
	"time"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
)

type ConversationPersistenceAdapter interface {
	NewConversationID(ctx context.Context) string
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

func NewRun(ctx context.Context, cm *CommonConversationManager, namespace string, previousRunID string, messages []responses.InputMessageUnion, options ...RunOption) (*ConversationRunManager, error) {
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

	// Run the options
	for _, o := range options {
		o(cr)
	}

	if cr.conversationId == "" {
		cr.conversationId = cr.ConversationPersistenceAdapter.NewConversationID(ctx)
	}

	return cr, nil
}

type RunOption func(manager *ConversationRunManager)

func WithConversationID(cid string) RunOption {
	return func(cm *ConversationRunManager) {
		cm.conversationId = cid
	}
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
	if cm.ConversationPersistenceAdapter == nil {
		return []responses.InputMessageUnion{}, nil
	}

	// Don't have to reload
	if len(cm.oldMessages) > 0 {
		return cm.oldMessages, nil
	}

	convMessages, err := cm.ConversationPersistenceAdapter.LoadMessages(ctx, namespace, previousMessageID)
	if err != nil {
		return nil, err
	}

	messages := []responses.InputMessageUnion{}
	for _, msg := range convMessages {
		for _, m := range msg.Messages {
			cm.msgIdToRunId[m.ID()] = msg.MessageID
		}
		cm.threadId = msg.ThreadID
		cm.conversationId = msg.ConversationID

		messages = append(messages, msg.Messages...)

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
	cm.RunState = core.LoadRunStateFromMeta(cm.lastMessageMeta)
	if cm.RunState != nil {
		cm.usage = &cm.RunState.Usage
	}

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

// GetConversationID GetOrCreateConversationID returns the conversation ID, if it doesn't exist it will create one
func (cm *ConversationRunManager) GetConversationID() string {
	return cm.conversationId
}

func (cm *ConversationRunManager) SaveMessages(ctx context.Context, meta map[string]any) error {
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
				return err
			}
		}

		cm.summaries = nil
	}

	if cm.ConversationPersistenceAdapter != nil {
		err := cm.ConversationPersistenceAdapter.SaveMessages(ctx, cm.namespace, cm.msgId, cm.previousMsgId, cm.conversationId, cm.newMessages, meta)
		if err != nil {
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
