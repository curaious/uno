package controllers

import (
	"errors"

	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/agent"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	"github.com/curaious/uno/internal/perrors"
)

func RegisterAgentRoutes(r *router.Router, svc *services.Services) {
	// Create agent
	r.POST("/api/agent-server/agents", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body agent.CreateAgentRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if body.ModelID == uuid.Nil {
			writeError(ctx, stdCtx, "Model ID is required", perrors.NewErrInvalidRequest("Model ID is required", errors.New("model_id is required")))
			return
		}

		if body.PromptID == uuid.Nil {
			writeError(ctx, stdCtx, "Prompt ID is required", perrors.NewErrInvalidRequest("Prompt ID is required", errors.New("prompt_id is required")))
			return
		}

		created, err := svc.Agent.Create(stdCtx, projectID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create agent", perrors.NewErrInternalServerError("Failed to create agent", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent created successfully", created)
	})

	// List agents
	r.GET("/api/agent-server/agents", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		agents, err := svc.Agent.List(stdCtx, projectID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list agents", perrors.NewErrInternalServerError("Failed to list agents", err))
			return
		}

		writeOK(ctx, stdCtx, "Agents retrieved successfully", agents)
	})

	// Get agent by ID
	r.GET("/api/agent-server/agents/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		idRaw, err := pathParam(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		id, err := uuid.Parse(idRaw)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		agt, err := svc.Agent.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent", perrors.NewErrInternalServerError("Failed to get agent", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent retrieved successfully", agt)
	})

	// Get agent by name
	r.GET("/api/agent-server/agents/by-name", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Agent name is required", perrors.NewErrInvalidRequest("Agent name is required", err))
			return
		}

		agt, err := svc.Agent.GetByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent", perrors.NewErrInternalServerError("Failed to get agent", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent retrieved successfully", agt)
	})

	// Update agent
	r.PUT("/api/agent-server/agents/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		idRaw, err := pathParam(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		id, err := uuid.Parse(idRaw)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		var body agent.UpdateAgentRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name != nil && *body.Name == "" {
			writeError(ctx, stdCtx, "Name cannot be empty", perrors.NewErrInvalidRequest("Name cannot be empty", errors.New("name cannot be empty")))
			return
		}

		updated, err := svc.Agent.Update(stdCtx, projectID, id, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update agent", perrors.NewErrInternalServerError("Failed to update agent", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent updated successfully", updated)
	})

	// Delete agent
	r.DELETE("/api/agent-server/agents/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		idRaw, err := pathParam(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		id, err := uuid.Parse(idRaw)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		if err := svc.Agent.Delete(stdCtx, projectID, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete agent", perrors.NewErrInternalServerError("Failed to delete agent", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent deleted successfully", nil)
	})
}
