package core

import (
	"context"

	"github.com/praveen001/uno/pkg/llm/responses"
)

type ChatHistory interface {
	// AddMessages adds new messages into a list in-memory
	AddMessages(ctx context.Context, messages []responses.InputMessageUnion, usage *responses.Usage)

	// GetMessages returns a list of messages from in-memory. It handles summarization if enabled
	GetMessages(ctx context.Context) ([]responses.InputMessageUnion, error)

	// LoadMessages fetches all the messages of the thread from persistent storage into memory
	LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]responses.InputMessageUnion, error)

	// SaveMessages saves the messages to persistent storage
	SaveMessages(ctx context.Context, meta map[string]any) error

	// GetMeta returns the meta from the most recent message
	GetMeta() map[string]any

	// GetMessageID returns the current run id
	GetMessageID() string
}
