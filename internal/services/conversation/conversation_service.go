package conversation

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ConversationService struct {
	repo *ConversationRepo
}

func NewConversationService(r *ConversationRepo) *ConversationService {
	return &ConversationService{
		repo: r,
	}
}

func (s *ConversationService) AddMessages(ctx context.Context, in *AddMessageRequest) error {
	// Case 1:
	// User is starting a new conversation
	if in.PreviousMessageID == "" {
		conversationID := in.ConversationID
		if conversationID == "" {
			conversationID = uuid.NewString()
		}

		// Create conversation
		conversation, err := s.repo.CreateConversation(ctx, Conversation{
			ProjectID:      in.ProjectID,
			NamespaceID:    in.Namespace,
			ConversationID: conversationID,
			Name:           "New Conversation",
			CreatedAt:      time.Now(),
			LastUpdated:    time.Now(),
		})
		if err != nil {
			return err
		}

		// Create a thread in the conversation
		thread, err := s.repo.CreateThread(ctx, Thread{
			ConversationID:  conversation.ConversationID,
			OriginMessageID: "",
			LastMessageID:   "",
			ThreadID:        uuid.NewString(),
			Meta:            in.Meta,
			CreatedAt:       time.Now(),
			LastUpdated:     time.Now(),
		})
		if err != nil {
			return err
		}

		// Add message to the thread
		err = s.repo.CreateMessages(ctx, ConversationMessage{
			ConversationID: conversation.ConversationID,
			ThreadID:       thread.ThreadID,
			MessageID:      in.MessageID,
			Messages:       in.Messages,
			Meta:           in.Meta,
		})
		if err != nil {
			return err
		}

		// Update the conversation's last updated timestamp
		conversation.LastUpdated = time.Now()
		err = s.repo.UpdateConversation(ctx, conversation)
		if err != nil {
			return err
		}

		// Update the thread's last message ID
		if len(in.Messages) > 0 {
			thread.LastMessageID = in.MessageID
			err = s.repo.UpdateThread(ctx, thread)
			if err != nil {
				return err
			}
		}

		return err
	}

	// Case 2:
	// User is continuing an existing conversation
	if in.PreviousMessageID != "" {
		// Fetch the message and its associated thread
		message, err := s.repo.GetMessageByID(ctx, in.ProjectID, in.Namespace, in.PreviousMessageID)
		if err != nil {
			return err
		}

		conversation, err := s.repo.GetConversationByID(ctx, in.ProjectID, in.Namespace, message.ConversationID)
		if err != nil {
			return err
		}

		thread, err := s.repo.GetThreadByID(ctx, in.ProjectID, in.Namespace, message.ThreadID)
		if err != nil {
			return err
		}

		if thread.LastMessageID == in.PreviousMessageID {
			// Append the new message to the existing thread
			err = s.repo.CreateMessages(ctx, ConversationMessage{
				MessageID:      in.MessageID,
				ThreadID:       thread.ThreadID,
				ConversationID: conversation.ConversationID,
				Messages:       in.Messages,
				Meta:           in.Meta,
			})
			if err != nil {
				return err
			}

			// Update the conversation's last updated timestamp
			conversation.LastUpdated = time.Now()
			err = s.repo.UpdateConversation(ctx, conversation)
			if err != nil {
				return err
			}

			// Update the thread's last message ID
			if len(in.Messages) > 0 {
				thread.LastMessageID = in.MessageID
				thread.LastUpdated = time.Now()
				thread.Meta = in.Meta
				err = s.repo.UpdateThread(ctx, thread)
				if err != nil {
					return err
				}
			}
		} else {
			// TODO: Create a new thread in the same conversation
		}

		return err
	}

	return nil
}

func (s *ConversationService) GetAllMessagesTillRun(ctx context.Context, in *GetMessagesRequest) ([]ConversationMessage, error) {
	convMessages, err := s.repo.GetAllMessagesTillRun(ctx, in.ProjectID, in.Namespace, in.PreviousMessageID)
	if err != nil {
		return nil, err
	}

	if len(convMessages) == 0 {
		return []ConversationMessage{}, nil
	}

	return convMessages, nil
}

func (s *ConversationService) ListConversations(ctx context.Context, projectID uuid.UUID, namespaceID string) ([]Conversation, error) {
	return s.repo.ListConversations(ctx, projectID, namespaceID)
}

func (s *ConversationService) ListThreads(ctx context.Context, projectID uuid.UUID, namespaceID string, conversationID string) ([]Thread, error) {
	return s.repo.ListThreads(ctx, projectID, namespaceID, conversationID)
}

func (s *ConversationService) ListMessages(ctx context.Context, projectID uuid.UUID, namespaceID string, threadID string) ([]ConversationMessage, error) {
	return s.repo.GetThreadMessages(ctx, projectID, namespaceID, threadID, 0, 100)
}

func (s *ConversationService) GetMessage(ctx context.Context, projectID uuid.UUID, namespaceID string, messageID string) (ConversationMessage, error) {
	return s.repo.GetMessageByID(ctx, projectID, namespaceID, messageID)
}

func (s *ConversationService) GetThread(ctx context.Context, projectID uuid.UUID, namespaceID string, threadID string) (Thread, error) {
	return s.repo.GetThreadByID(ctx, projectID, namespaceID, threadID)
}

func (s *ConversationService) GetConversation(ctx context.Context, projectID uuid.UUID, namespaceID string, conversationID string) (Conversation, error) {
	return s.repo.GetConversationByID(ctx, projectID, namespaceID, conversationID)
}

func (s *ConversationService) CreateSummary(ctx context.Context, projectID uuid.UUID, namespace string, summary Summary) error {
	return s.repo.CreateSummary(ctx, summary)
}
