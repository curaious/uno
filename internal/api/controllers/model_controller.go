package controllers

import (
	"errors"

	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/internal/services/model"
	"github.com/valyala/fasthttp"

	"github.com/praveen001/uno/internal/perrors"
)

func RegisterModelRoutes(r *router.Router, svc *services.Services) {
	// Create model
	r.POST("/api/agent-server/models", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body model.CreateModelRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if body.ModelID == "" {
			writeError(ctx, stdCtx, "Model ID is required", perrors.NewErrInvalidRequest("Model ID is required", errors.New("model_id is required")))
			return
		}

		if body.ProviderType == "" {
			writeError(ctx, stdCtx, "Provider type is required", perrors.NewErrInvalidRequest("Provider type is required", errors.New("provider_type is required")))
			return
		}

		m, err := svc.Model.Create(stdCtx, projectID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create model", perrors.NewErrInternalServerError("Failed to create model", err))
			return
		}

		writeOK(ctx, stdCtx, "Model created successfully", m)
	})

	// List models
	r.GET("/api/agent-server/models", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		providerTypeParam := string(ctx.QueryArgs().Peek("provider_type"))
		var providerType *string
		if providerTypeParam != "" {
			pt := providerTypeParam
			providerType = &pt
		}

		models, err := svc.Model.List(stdCtx, projectID, providerType)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list models", perrors.NewErrInternalServerError("Failed to list models", err))
			return
		}

		writeOK(ctx, stdCtx, "Models retrieved successfully", models)
	})

	// Get model by ID
	r.GET("/api/agent-server/models/{id}", func(ctx *fasthttp.RequestCtx) {
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

		m, err := svc.Model.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get model", perrors.NewErrInternalServerError("Failed to get model", err))
			return
		}

		writeOK(ctx, stdCtx, "Model retrieved successfully", m)
	})

	// Get model by name
	r.GET("/api/agent-server/models/by-name", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", err))
			return
		}

		m, err := svc.Model.GetByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get model", perrors.NewErrInternalServerError("Failed to get model", err))
			return
		}

		writeOK(ctx, stdCtx, "Model retrieved successfully", m)
	})

	// Update model
	r.PUT("/api/agent-server/models/{id}", func(ctx *fasthttp.RequestCtx) {
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

		var body model.UpdateModelRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name != nil && *body.Name == "" {
			writeError(ctx, stdCtx, "Name cannot be empty", perrors.NewErrInvalidRequest("Name cannot be empty", errors.New("name cannot be empty")))
			return
		}

		if body.ModelID != nil && *body.ModelID == "" {
			writeError(ctx, stdCtx, "Model ID cannot be empty", perrors.NewErrInvalidRequest("Model ID cannot be empty", errors.New("model_id cannot be empty")))
			return
		}

		m, err := svc.Model.Update(stdCtx, projectID, id, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update model", perrors.NewErrInternalServerError("Failed to update model", err))
			return
		}

		writeOK(ctx, stdCtx, "Model updated successfully", m)
	})

	// Delete model
	r.DELETE("/api/agent-server/models/{id}", func(ctx *fasthttp.RequestCtx) {
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

		if err := svc.Model.Delete(stdCtx, projectID, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete model", perrors.NewErrInternalServerError("Failed to delete model", err))
			return
		}

		writeOK(ctx, stdCtx, "Model deleted successfully", nil)
	})
}
