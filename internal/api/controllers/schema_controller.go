package controllers

import (
	"errors"

	"github.com/curaious/uno/internal/perrors"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/schema"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func RegisterSchemaRoutes(r *router.Router, svc *services.Services) {
	// Create schema
	r.POST("/api/agent-server/schemas", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body schema.CreateSchemaRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if body.Schema == nil || len(body.Schema) == 0 {
			writeError(ctx, stdCtx, "Schema is required", perrors.NewErrInvalidRequest("Schema is required", errors.New("schema is required")))
			return
		}

		s, err := svc.Schema.CreateSchema(stdCtx, projectID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create schema", perrors.NewErrInternalServerError("Failed to create schema", err))
			return
		}

		writeOK(ctx, stdCtx, "Schema created successfully", s)
	})

	// List schemas
	r.GET("/api/agent-server/schemas", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		schemas, err := svc.Schema.ListSchemas(stdCtx, projectID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list schemas", perrors.NewErrInternalServerError("Failed to list schemas", err))
			return
		}

		writeOK(ctx, stdCtx, "Schemas retrieved successfully", schemas)
	})

	// Get schema by ID
	r.GET("/api/agent-server/schemas/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		id, err := pathParamUUID(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid schema ID", perrors.NewErrInvalidRequest("Invalid schema ID", err))
			return
		}

		s, err := svc.Schema.GetSchema(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get schema", perrors.NewErrInternalServerError("Failed to get schema", err))
			return
		}

		writeOK(ctx, stdCtx, "Schema retrieved successfully", s)
	})

	// Get schema by name
	r.GET("/api/agent-server/schemas/by-name", func(ctx *fasthttp.RequestCtx) {
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

		s, err := svc.Schema.GetSchemaByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get schema", perrors.NewErrInternalServerError("Failed to get schema", err))
			return
		}

		writeOK(ctx, stdCtx, "Schema retrieved successfully", s)
	})

	// Update schema
	r.PUT("/api/agent-server/schemas/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		id, err := pathParamUUID(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid schema ID", perrors.NewErrInvalidRequest("Invalid schema ID", err))
			return
		}

		var body schema.UpdateSchemaRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		s, err := svc.Schema.UpdateSchema(stdCtx, projectID, id, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update schema", perrors.NewErrInternalServerError("Failed to update schema", err))
			return
		}

		writeOK(ctx, stdCtx, "Schema updated successfully", s)
	})

	// Delete schema
	r.DELETE("/api/agent-server/schemas/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		id, err := pathParamUUID(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid schema ID", perrors.NewErrInvalidRequest("Invalid schema ID", err))
			return
		}

		if err := svc.Schema.DeleteSchema(stdCtx, projectID, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete schema", perrors.NewErrInternalServerError("Failed to delete schema", err))
			return
		}

		writeOK(ctx, stdCtx, "Schema deleted successfully", nil)
	})
}
