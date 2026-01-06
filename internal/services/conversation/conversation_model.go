package conversation

import (
	"time"

	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
)

// Conversation represents a conversation
type Conversation struct {
	ProjectID      uuid.UUID `json:"project_id" db:"project_id"`
	NamespaceID    string    `json:"namespace_id" db:"namespace_id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	Name           string    `json:"name" db:"name"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	LastUpdated    time.Time `json:"last_updated" db:"last_updated"`
}

// Thread represents a thread within a conversation
type Thread struct {
	ConversationID  string                 `json:"conversation_id" db:"conversation_id"`
	OriginMessageID string                 `json:"origin_message_id" db:"origin_message_id"`
	LastMessageID   string                 `json:"last_message_id" db:"last_message_id"`
	ThreadID        string                 `json:"thread_id" db:"thread_id"`
	Meta            map[string]interface{} `json:"meta" db:"meta"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	LastUpdated     time.Time              `json:"last_updated" db:"last_updated"`
}

// ConversationMessage represents a message within a thread
type ConversationMessage struct {
	MessageID      string                        `json:"message_id" db:"message_id"`
	ThreadID       string                        `json:"thread_id" db:"thread_id"`
	ConversationID string                        `json:"conversation_id" db:"conversation_id"`
	Messages       []responses.InputMessageUnion `json:"messages" db:"messages"`
	Meta           map[string]any                `json:"meta" db:"meta"`
}

// Summary represents a conversation summary stored in the summaries table
type Summary struct {
	ID                      string                      `json:"id" db:"id"`
	ThreadID                string                      `json:"thread_id" db:"thread_id"`
	SummaryMessage          responses.InputMessageUnion `json:"summary_message" db:"summary_message"`
	LastSummarizedMessageID string                      `json:"last_summarized_message_id" db:"last_summarized_message_id"`
	CreatedAt               time.Time                   `json:"created_at" db:"created_at"`
	Meta                    map[string]any              `json:"meta" db:"meta"`
}

type AddMessageRequest struct {
	ProjectID         uuid.UUID                     `json:"project_id"`
	Namespace         string                        `json:"namespace"`
	MessageID         string                        `json:"message_id"`
	PreviousMessageID string                        `json:"previous_message_id"`
	Messages          []responses.InputMessageUnion `json:"messages"`
	Meta              map[string]any                `json:"meta"`
	ConversationID    string                        `json:"conversation_id"`
}

type GetMessagesRequest struct {
	ProjectID         uuid.UUID `json:"project_id"`
	Namespace         string    `json:"namespace"`
	PreviousMessageID string    `json:"previous_message_id"`
	Offset            int       `json:"offset"`
	Limit             int       `json:"limit"`
}
