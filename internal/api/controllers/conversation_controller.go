package controllers

import (
	"github.com/fasthttp/router"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/internal/services/conversation"
	"github.com/valyala/fasthttp"

	"github.com/praveen001/uno/internal/perrors"
)

func RegisterConversationRoutes(r *router.Router, svc *services.Services) {
	// List conversations
	r.GET("/api/agent-server/conversations", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		conversations, err := svc.Conversation.ListConversations(stdCtx, projectID, namespace)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list conversations", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", conversations)
	})

	// List threads in a conversation
	r.GET("/api/agent-server/threads", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		conversationID, err := requireStringQuery(ctx, "conversation_id")
		if err != nil {
			writeError(ctx, stdCtx, "Conversation ID is required", perrors.NewErrInvalidRequest("Conversation ID is required", err))
			return
		}
		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		threads, err := svc.Conversation.ListThreads(stdCtx, projectID, namespace, conversationID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list threads", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", threads)
	})

	// List messages in a thread
	r.GET("/api/agent-server/messages", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		threadID, err := requireStringQuery(ctx, "thread_id")
		if err != nil {
			writeError(ctx, stdCtx, "Thread ID is required", perrors.NewErrInvalidRequest("Thread ID is required", err))
			return
		}

		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}

		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		messages, err := svc.Conversation.ListMessages(stdCtx, projectID, namespace, threadID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list messages", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", messages)
	})

	// Get specific message by ID
	r.GET("/api/agent-server/messages/{message_id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		messageID, err := pathParam(ctx, "message_id")
		if err != nil {
			writeError(ctx, stdCtx, "Message ID is required", perrors.NewErrInvalidRequest("Message ID is required", err))
			return
		}

		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}

		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		message, err := svc.Conversation.GetMessage(stdCtx, projectID, namespace, messageID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get message", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", message)
	})

	// Add messages to a conversation/thread
	r.POST("/api/agent-server/messages", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body conversation.AddMessageRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		body.ProjectID = projectID
		if err := svc.Conversation.AddMessages(stdCtx, &body); err != nil {
			writeError(ctx, stdCtx, "Failed to add messages", err)
			return
		}

		writeOK(ctx, stdCtx, "Messages added successfully", nil)
	})

	// Get all messages till a specific run (previous_message_id)
	r.GET("/api/agent-server/messages/summary", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}

		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		previousMessageID := string(ctx.QueryArgs().Peek("previous_message_id"))

		messages, err := svc.Conversation.GetAllMessagesTillRun(stdCtx, &conversation.GetMessagesRequest{
			ProjectID:         projectID,
			Namespace:         namespace,
			PreviousMessageID: previousMessageID,
		})
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get messages summary", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", messages)
	})

	// Get specific thread by ID
	r.GET("/api/agent-server/threads/{thread_id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		threadID, err := pathParam(ctx, "thread_id")
		if err != nil {
			writeError(ctx, stdCtx, "Thread ID is required", perrors.NewErrInvalidRequest("Thread ID is required", err))
			return
		}

		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}

		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		thread, err := svc.Conversation.GetThread(stdCtx, projectID, namespace, threadID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get thread", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", thread)
	})

	// Get specific conversation by ID
	r.GET("/api/agent-server/conversations/{conversation_id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		conversationID, err := pathParam(ctx, "conversation_id")
		if err != nil {
			writeError(ctx, stdCtx, "Conversation ID is required", perrors.NewErrInvalidRequest("Conversation ID is required", err))
			return
		}

		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}

		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		conv, err := svc.Conversation.GetConversation(stdCtx, projectID, namespace, conversationID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get conversation", err)
			return
		}

		writeOK(ctx, stdCtx, "OK", conv)
	})

	// Save summary
	r.POST("/api/agent-server/summary", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		namespace, err := requireStringQuery(ctx, "namespace")
		if err != nil {
			writeError(ctx, stdCtx, "Namespace is required", perrors.NewErrInvalidRequest("Namespace is required", err))
			return
		}

		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body conversation.Summary
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if err := svc.Conversation.CreateSummary(stdCtx, projectID, namespace, body); err != nil {
			writeError(ctx, stdCtx, "Failed to create summary", err)
			return
		}

		writeOK(ctx, stdCtx, "Summary saved successfully", nil)
	})
}
