package controllers

import (
	"errors"

	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services"
	project2 "github.com/praveen001/uno/internal/services/project"
	"github.com/valyala/fasthttp"

	"github.com/praveen001/uno/internal/perrors"
)

func RegisterProjectRoutes(r *router.Router, svc *services.Services) {
	// Create project
	r.POST("/api/agent-server/projects", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		var body project2.CreateProjectRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		created, err := svc.Project.Create(stdCtx, &body)
		if err != nil {
			switch {
			case errors.Is(err, project2.ErrProjectAlreadyExists):
				writeError(ctx, stdCtx, "Project with this name already exists", perrors.New(perrors.ErrCodeConflict, "Project with this name already exists", err))
			default:
				writeError(ctx, stdCtx, "Failed to create project", perrors.NewErrInternalServerError("Failed to create project", err))
			}
			return
		}

		writeOK(ctx, stdCtx, "Project created successfully", created)
	})

	// List projects
	r.GET("/api/agent-server/projects", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projects, err := svc.Project.List(stdCtx)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list projects", perrors.NewErrInternalServerError("Failed to list projects", err))
			return
		}

		writeOK(ctx, stdCtx, "Projects retrieved successfully", projects)
	})

	// Get project by name
	r.GET("/api/agent-server/project/id", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", err))
			return
		}

		p, err := svc.Project.GetByName(stdCtx, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list projects", perrors.NewErrInternalServerError("Failed to list projects", err))
			return
		}

		writeOK(ctx, stdCtx, "Projects retrieved successfully", p.ID.String())
	})

	// Update project
	r.PUT("/api/agent-server/projects/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
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

		var body project2.UpdateProjectRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name != nil && *body.Name == "" {
			writeError(ctx, stdCtx, "Name cannot be empty", perrors.NewErrInvalidRequest("Name cannot be empty", errors.New("name cannot be empty")))
			return
		}

		updated, err := svc.Project.Update(stdCtx, id, &body)
		if err != nil {
			switch {
			case errors.Is(err, project2.ErrProjectNotFound):
				writeError(ctx, stdCtx, "Project not found", perrors.New(perrors.ErrCodeNotFound, "Project not found", err))
			case errors.Is(err, project2.ErrProjectAlreadyExists):
				writeError(ctx, stdCtx, "Project with this name already exists", perrors.New(perrors.ErrCodeConflict, "Project with this name already exists", err))
			default:
				writeError(ctx, stdCtx, "Failed to update project", perrors.NewErrInternalServerError("Failed to update project", err))
			}
			return
		}

		writeOK(ctx, stdCtx, "Project updated successfully", updated)
	})

	// Delete project
	r.DELETE("/api/agent-server/projects/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
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

		if err := svc.Project.Delete(stdCtx, id); err != nil {
			switch {
			case errors.Is(err, project2.ErrProjectNotFound):
				writeError(ctx, stdCtx, "Project not found", perrors.New(perrors.ErrCodeNotFound, "Project not found", err))
			default:
				writeError(ctx, stdCtx, "Failed to delete project", perrors.NewErrInternalServerError("Failed to delete project", err))
			}
			return
		}

		writeOK(ctx, stdCtx, "Project deleted successfully", nil)
	})
}
