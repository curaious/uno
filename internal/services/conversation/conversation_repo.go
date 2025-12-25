package conversation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type ConversationRepo struct {
	db *sqlx.DB
}

func NewConversationRepo(db *sqlx.DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	query := `
		INSERT INTO conversations (project_id, namespace_id, conversation_id, name, created_at, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING project_id, namespace_id, conversation_id, name, created_at, last_updated
	`

	var result Conversation
	err := r.db.GetContext(ctx, &result, query,
		conversation.ProjectID,
		conversation.NamespaceID,
		conversation.ConversationID,
		conversation.Name,
		conversation.CreatedAt,
		conversation.LastUpdated,
	)

	return result, err
}

func (r *ConversationRepo) UpdateConversation(ctx context.Context, conversation Conversation) error {
	query := `
		UPDATE conversations 
		SET name = $1, last_updated = $2
		WHERE conversation_id = $3 AND namespace_id = $4 AND project_id = $5
	`

	result, err := r.db.ExecContext(ctx, query,
		conversation.Name,
		conversation.LastUpdated,
		conversation.ConversationID,
		conversation.NamespaceID,
		conversation.ProjectID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ConversationRepo) GetConversationByID(ctx context.Context, projectID uuid.UUID, namespace string, conversationID string) (Conversation, error) {
	query := `
		SELECT project_id, namespace_id, conversation_id, name, created_at, last_updated
		FROM conversations
		WHERE conversation_id = $1 AND namespace_id = $2 AND project_id = $3
	`

	var conversation Conversation
	err := r.db.GetContext(ctx, &conversation, query, conversationID, namespace, projectID)
	return conversation, err
}

func (r *ConversationRepo) CreateThread(ctx context.Context, thread Thread) (Thread, error) {
	query := `
		INSERT INTO threads (conversation_id, origin_message_id, thread_id, meta, created_at, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING conversation_id, origin_message_id, thread_id, meta, created_at, last_updated
	`

	metaJSON, err := json.Marshal(thread.Meta)
	if err != nil {
		return thread, err
	}

	_, err = r.db.ExecContext(ctx, query,
		thread.ConversationID,
		thread.OriginMessageID,
		thread.ThreadID,
		metaJSON,
		thread.CreatedAt,
		thread.LastUpdated,
	)

	return thread, err
}

func (r *ConversationRepo) CreateMessages(ctx context.Context, message ConversationMessage) error {
	if len(message.Messages) == 0 {
		return nil
	}

	query := `
		INSERT INTO messages (id, thread_id, conversation_id, messages, meta, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) 
		ON CONFLICT (id) DO UPDATE
		SET
    		messages = messages.messages || EXCLUDED.messages,
    		meta     = EXCLUDED.meta;
	`

	stmt, err := r.db.PreparexContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	messagesJSON, err := json.Marshal(message.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal message content: %w", err)
	}

	metaJSON, err := json.Marshal(message.Meta)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx,
		message.MessageID,
		message.ThreadID,
		message.ConversationID,
		messagesJSON,
		metaJSON,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert message %s: %w", message.MessageID, err)
	}

	return nil
}

func (r *ConversationRepo) GetMessageByID(ctx context.Context, projectID uuid.UUID, namespace string, ID string) (ConversationMessage, error) {
	query := `
		SELECT m.id as message_id, m.thread_id, t.conversation_id, m.messages, m.meta
		FROM messages m
		JOIN threads t ON m.thread_id = t.thread_id
		JOIN conversations c ON t.conversation_id = c.conversation_id
		WHERE m.id = $1 AND c.namespace_id = $2 AND c.project_id = $3
	`

	var result struct {
		MessageID      string           `db:"message_id"`
		ThreadID       string           `db:"thread_id"`
		ConversationID string           `db:"conversation_id"`
		Messages       utils.RawMessage `db:"messages"`
		Meta           utils.RawMessage `db:"meta"`
	}

	err := r.db.GetContext(ctx, &result, query, ID, namespace, projectID)
	if err != nil {
		return ConversationMessage{}, err
	}

	var messages []responses.InputMessageUnion
	err = json.Unmarshal(result.Messages, &messages)
	if err != nil {
		return ConversationMessage{}, fmt.Errorf("failed to unmarshal message content: %w", err)
	}

	var meta map[string]any
	err = json.Unmarshal(result.Meta, &meta)
	if err != nil {
		meta = make(map[string]any)
	}

	message := ConversationMessage{
		MessageID:      result.MessageID,
		ThreadID:       result.ThreadID,
		ConversationID: result.ConversationID,
		Messages:       messages,
		Meta:           meta,
	}

	return message, nil
}

func (r *ConversationRepo) GetThreadByID(ctx context.Context, projectID uuid.UUID, namespace string, threadID string) (Thread, error) {
	query := `
		SELECT t.conversation_id, t.origin_message_id, t.last_message_id, t.thread_id, t.meta, t.created_at, t.last_updated
		FROM threads t
		JOIN conversations c ON t.conversation_id = c.conversation_id
		WHERE t.thread_id = $1 AND c.namespace_id = $2 AND c.project_id = $3
	`

	thread := Thread{}
	results, err := r.db.QueryContext(ctx, query, threadID, namespace, projectID)
	defer results.Close()

	for results.Next() {
		var metaJSON []byte

		err = results.Scan(
			&thread.ConversationID,
			&thread.OriginMessageID,
			&thread.LastMessageID,
			&thread.ThreadID,
			&metaJSON,
			&thread.CreatedAt,
			&thread.LastUpdated,
		)
		if err != nil {
			return thread, err
		}

		// Unmarshal meta JSON
		if len(metaJSON) > 0 {
			err = json.Unmarshal(metaJSON, &thread.Meta)
			if err != nil {
				thread.Meta = make(map[string]interface{})
			}
		} else {
			thread.Meta = make(map[string]interface{})
		}

		return thread, nil
	}

	return thread, err
}

func (r *ConversationRepo) UpdateThread(ctx context.Context, thread Thread) error {
	query := `
		UPDATE threads 
		SET origin_message_id = $1, last_message_id = $2, meta = $3, last_updated = $4
		WHERE thread_id = $5
	`

	metaJSON, err := json.Marshal(thread.Meta)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, query,
		thread.OriginMessageID,
		thread.LastMessageID,
		metaJSON,
		thread.LastUpdated,
		thread.ThreadID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *ConversationRepo) ListConversations(ctx context.Context, projectID uuid.UUID, namespaceID string) ([]Conversation, error) {
	query := `
		SELECT project_id, namespace_id, conversation_id, name, created_at, last_updated
		FROM conversations
		WHERE namespace_id = $1 AND project_id = $2
		ORDER BY last_updated DESC
	`

	conversations := []Conversation{}
	err := r.db.SelectContext(ctx, &conversations, query, namespaceID, projectID)
	return conversations, err
}

func (r *ConversationRepo) ListThreads(ctx context.Context, projectID uuid.UUID, namespaceID string, conversationID string) ([]Thread, error) {
	query := `
		SELECT t.conversation_id, t.origin_message_id, t.last_message_id, t.thread_id, t.meta, t.created_at, t.last_updated
		FROM threads t
		JOIN conversations c ON t.conversation_id = c.conversation_id
		WHERE c.namespace_id = $1 AND t.conversation_id = $2 AND c.project_id = $3
		ORDER BY t.last_updated DESC
	`

	rows, err := r.db.QueryContext(ctx, query, namespaceID, conversationID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	threads := []Thread{}
	for rows.Next() {
		var thread Thread
		var metaJSON []byte

		err := rows.Scan(
			&thread.ConversationID,
			&thread.OriginMessageID,
			&thread.LastMessageID,
			&thread.ThreadID,
			&metaJSON,
			&thread.CreatedAt,
			&thread.LastUpdated,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal meta JSON
		if len(metaJSON) > 0 {
			err = json.Unmarshal(metaJSON, &thread.Meta)
			if err != nil {
				// If unmarshaling fails, initialize as empty map
				thread.Meta = make(map[string]interface{})
			}
		} else {
			thread.Meta = make(map[string]interface{})
		}

		threads = append(threads, thread)
	}

	return threads, err
}

func (r *ConversationRepo) GetThreadMessages(ctx context.Context, projectID uuid.UUID, namespace string, threadID string, offset, limit int) ([]ConversationMessage, error) {
	query := `
		SELECT m.id as message_id, m.thread_id, t.conversation_id, m.messages, m.meta
		FROM messages m
		JOIN threads t ON m.thread_id = t.thread_id
		JOIN conversations c ON t.conversation_id = c.conversation_id
		WHERE m.thread_id = $1 AND c.namespace_id = $2 AND c.project_id = $3
		ORDER BY m.created_at ASC
		LIMIT $4 OFFSET $5
	`

	messages := []ConversationMessage{}
	results, err := r.db.QueryContext(ctx, query, threadID, namespace, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer results.Close()

	for results.Next() {
		message := ConversationMessage{}
		rawMessages := []byte{}
		rawMeta := []byte{}

		err = results.Scan(&message.MessageID, &message.ThreadID, &message.ConversationID, &rawMessages, &rawMeta)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(rawMessages, &message.Messages)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}

		err = json.Unmarshal(rawMeta, &message.Meta)
		if err != nil {
			message.Meta = make(map[string]interface{})
		}

		messages = append(messages, message)

	}

	return messages, nil
}

func (r *ConversationRepo) GetAllMessagesTillRun(ctx context.Context, projectID uuid.UUID, namespace string, previousMessageID string) ([]ConversationMessage, error) {
	if previousMessageID == "" {
		return []ConversationMessage{}, nil
	}

	// First, find the thread ID for the given previous message ID
	var thread Thread
	queryThread := `
		SELECT m.thread_id, m.conversation_id
		FROM messages m
		JOIN threads t ON m.thread_id = t.thread_id
		JOIN conversations c ON t.conversation_id = c.conversation_id
		WHERE m.id = $1 AND c.namespace_id = $2 AND c.project_id = $3
	`

	err := r.db.GetContext(ctx, &thread, queryThread, previousMessageID, namespace, projectID)
	if err != nil {
		return nil, err
	}

	// Optimization: Try to load from latest summary in summaries table
	// Check for summary preceding the previous message
	summary, err := r.GetLatestSummaryBeforeMessage(ctx, projectID, namespace, thread.ThreadID, previousMessageID)
	if err == nil && summary.ID != "" {
		// Found summary. Fetch messages between the summarized point and the previous message
		msgsBetween, err := r.getMessagesBetween(ctx, projectID, namespace, thread.ThreadID, summary.LastSummarizedMessageID, previousMessageID)
		if err == nil {
			// Convert summary to ConversationMessage format and combine with messages between
			summaryMsg := ConversationMessage{
				MessageID:      summary.ID,
				ThreadID:       summary.ThreadID,
				ConversationID: thread.ConversationID,
				Messages:       []responses.InputMessageUnion{summary.SummaryMessage},
				Meta:           summary.Meta,
			}
			return append([]ConversationMessage{summaryMsg}, msgsBetween...), nil
		}
	}

	// Fallback: fetch messages in that thread
	const limit = 100
	const offset = 0
	query := `
		SELECT m.id as message_id, m.thread_id, t.conversation_id, m.messages, m.meta
		FROM messages m
		JOIN threads t ON m.thread_id = t.thread_id
		JOIN conversations c ON t.conversation_id = c.conversation_id
		WHERE m.thread_id = $1 AND c.namespace_id = $2 AND c.project_id = $3
		ORDER BY m.created_at ASC
		LIMIT $4 OFFSET $5
	`

	messages := []ConversationMessage{}
	results, err := r.db.QueryContext(ctx, query, thread.ThreadID, namespace, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer results.Close()

	for results.Next() {
		message := ConversationMessage{}
		rawMessages := []byte{}
		rawMeta := []byte{}

		err = results.Scan(&message.MessageID, &message.ThreadID, &message.ConversationID, &rawMessages, &rawMeta)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(rawMessages, &message.Messages)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}

		err = json.Unmarshal(rawMeta, &message.Meta)
		if err != nil {
			message.Meta = make(map[string]interface{})
		}

		messages = append(messages, message)

		if message.MessageID == previousMessageID {
			break
		}
	}

	return messages, nil
}

func (r *ConversationRepo) getMessagesBetween(ctx context.Context, projectID uuid.UUID, namespace string, threadID string, startMessageID string, endMessageID string) ([]ConversationMessage, error) {
	query := `
		SELECT m.id as message_id, m.thread_id, t.conversation_id, m.messages, m.meta
		FROM messages m
		JOIN threads t ON m.thread_id = t.thread_id
		JOIN conversations c ON t.conversation_id = c.conversation_id
		JOIN messages start_ref ON start_ref.id = $4
		JOIN messages end_ref ON end_ref.id = $5
		WHERE m.thread_id = $1 AND c.namespace_id = $2 AND c.project_id = $3
		AND m.created_at > start_ref.created_at
		AND m.created_at <= end_ref.created_at
		ORDER BY m.created_at ASC
	`

	messages := []ConversationMessage{}
	results, err := r.db.QueryContext(ctx, query, threadID, namespace, projectID, startMessageID, endMessageID)
	if err != nil {
		return nil, err
	}
	defer results.Close()

	for results.Next() {
		message := ConversationMessage{}
		rawMessages := []byte{}
		rawMeta := []byte{}

		err = results.Scan(&message.MessageID, &message.ThreadID, &message.ConversationID, &rawMessages, &rawMeta)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(rawMessages, &message.Messages)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}

		err = json.Unmarshal(rawMeta, &message.Meta)
		if err != nil {
			message.Meta = make(map[string]interface{})
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// CreateSummary saves a summary to the summaries table
func (r *ConversationRepo) CreateSummary(ctx context.Context, summary Summary) error {
	query := `
		INSERT INTO summaries (id, thread_id, summary_message, last_summarized_message_id, created_at, meta)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	summaryJSON, err := json.Marshal(summary.SummaryMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal summary message: %w", err)
	}

	metaJSON, err := json.Marshal(summary.Meta)
	if err != nil {
		return fmt.Errorf("failed to marshal summary meta: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		summary.ID,
		summary.ThreadID,
		summaryJSON,
		summary.LastSummarizedMessageID,
		summary.CreatedAt,
		metaJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert summary %s: %w", summary.ID, err)
	}

	return nil
}

// GetLatestSummaryBeforeMessage finds the latest summary for a thread that precedes the given message ID
func (r *ConversationRepo) GetLatestSummaryBeforeMessage(ctx context.Context, projectID uuid.UUID, namespace string, threadID string, beforeMessageID string) (Summary, error) {
	query := `
		SELECT s.id, s.thread_id, s.summary_message, s.last_summarized_message_id, s.created_at, s.meta
		FROM summaries s
		JOIN threads t ON s.thread_id = t.thread_id
		JOIN conversations c ON t.conversation_id = c.conversation_id
		JOIN messages ref ON ref.id = $4
		WHERE s.thread_id = $1 AND c.namespace_id = $2 AND c.project_id = $3
		AND s.created_at <= ref.created_at
		ORDER BY s.created_at DESC
		LIMIT 1
	`

	var result struct {
		ID                      string           `db:"id"`
		ThreadID                string           `db:"thread_id"`
		SummaryMessage          utils.RawMessage `db:"summary_message"`
		LastSummarizedMessageID string           `db:"last_summarized_message_id"`
		CreatedAt               time.Time        `db:"created_at"`
		Meta                    utils.RawMessage `db:"meta"`
	}

	err := r.db.GetContext(ctx, &result, query, threadID, namespace, projectID, beforeMessageID)
	if err != nil {
		return Summary{}, err
	}

	var summaryMessage responses.InputMessageUnion
	err = json.Unmarshal(result.SummaryMessage, &summaryMessage)
	if err != nil {
		return Summary{}, fmt.Errorf("failed to unmarshal summary message: %w", err)
	}

	var meta map[string]any
	err = json.Unmarshal(result.Meta, &meta)
	if err != nil {
		meta = make(map[string]any)
	}

	return Summary{
		ID:                      result.ID,
		ThreadID:                result.ThreadID,
		SummaryMessage:          summaryMessage,
		LastSummarizedMessageID: result.LastSummarizedMessageID,
		CreatedAt:               result.CreatedAt,
		Meta:                    meta,
	}, nil
}
