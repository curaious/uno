package controllers

import (
	"errors"
	"strconv"

	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/prompt"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/curaious/uno/internal/perrors"
)

func RegisterPromptRoutes(r *router.Router, svc *services.Services) {
	// Create prompt
	r.POST("/api/agent-server/prompts", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body prompt.CreatePromptRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if body.Template == "" {
			writeError(ctx, stdCtx, "Template is required", perrors.NewErrInvalidRequest("Template is required", errors.New("template is required")))
			return
		}

		if body.CommitMessage == "" {
			writeError(ctx, stdCtx, "Commit message is required", perrors.NewErrInvalidRequest("Commit message is required", errors.New("commit_message is required")))
			return
		}

		p, err := svc.Prompt.CreatePrompt(stdCtx, projectID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create prompt", perrors.NewErrInternalServerError("Failed to create prompt", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt created successfully with first version", p)
	})

	// List prompts
	r.GET("/api/agent-server/prompts", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		prompts, err := svc.Prompt.ListPrompts(stdCtx, projectID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list prompts", perrors.NewErrInternalServerError("Failed to list prompts", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompts retrieved successfully", prompts)
	})

	// Get prompt by name
	r.GET("/api/agent-server/prompts/by-name", func(ctx *fasthttp.RequestCtx) {
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

		p, err := svc.Prompt.GetPrompt(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get prompt", perrors.NewErrInternalServerError("Failed to get prompt", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt retrieved successfully", p)
	})

	// Delete prompt
	r.DELETE("/api/agent-server/prompts", func(ctx *fasthttp.RequestCtx) {
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

		if err := svc.Prompt.DeletePrompt(stdCtx, projectID, name); err != nil {
			writeError(ctx, stdCtx, "Failed to delete prompt", perrors.NewErrInternalServerError("Failed to delete prompt", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt deleted successfully", nil)
	})

	// Create prompt version
	r.POST("/api/agent-server/prompts/{name}/versions", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := pathParam(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Prompt name is required", perrors.NewErrInvalidRequest("Prompt name is required", err))
			return
		}

		var body prompt.CreatePromptVersionRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Template == "" {
			writeError(ctx, stdCtx, "Template is required", perrors.NewErrInvalidRequest("Template is required", errors.New("template is required")))
			return
		}

		if body.CommitMessage == "" {
			writeError(ctx, stdCtx, "Commit message is required", perrors.NewErrInvalidRequest("Commit message is required", errors.New("commit_message is required")))
			return
		}

		// Validate label if provided
		if body.Label != nil && *body.Label != "" {
			validLabel := *body.Label == "production" || *body.Label == "latest"
			if !validLabel {
				writeError(ctx, stdCtx, "Label must be either 'production' or 'latest'", perrors.NewErrInvalidRequest("Label must be either 'production' or 'latest'", errors.New("invalid label value")))
				return
			}
		}

		version, err := svc.Prompt.CreatePromptVersion(stdCtx, projectID, name, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create prompt version", perrors.NewErrInternalServerError("Failed to create prompt version", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt version created successfully", version)
	})

	// List prompt versions
	r.GET("/api/agent-server/prompts/{name}/versions", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := pathParam(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Prompt name is required", perrors.NewErrInvalidRequest("Prompt name is required", err))
			return
		}

		versions, err := svc.Prompt.ListPromptVersions(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list prompt versions", perrors.NewErrInternalServerError("Failed to list prompt versions", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt versions retrieved successfully", versions)
	})

	// Get prompt version
	r.GET("/api/agent-server/prompts/{name}/versions/{version}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := pathParam(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Prompt name is required", perrors.NewErrInvalidRequest("Prompt name is required", err))
			return
		}

		versionStr, err := pathParam(ctx, "version")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		promptVersion, err := svc.Prompt.GetPromptVersion(stdCtx, projectID, name, version)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get prompt version", perrors.NewErrInternalServerError("Failed to get prompt version", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt version retrieved successfully", promptVersion)
	})

	// Get prompt version by label
	r.GET("/api/agent-server/prompts/{name}/label/{label}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := pathParam(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Prompt name is required", perrors.NewErrInvalidRequest("Prompt name is required", err))
			return
		}

		label, err := pathParam(ctx, "label")
		if err != nil {
			writeError(ctx, stdCtx, "Label is required", perrors.NewErrInvalidRequest("Label is required", err))
			return
		}

		promptVersion, err := svc.Prompt.GetPromptVersionByLabel(stdCtx, projectID, name, label)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get prompt version by label", perrors.NewErrInternalServerError("Failed to get prompt version by label", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt version retrieved successfully", promptVersion)
	})

	// Update prompt version label
	r.PATCH("/api/agent-server/prompts/{name}/versions/{version}/label", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := pathParam(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Prompt name is required", perrors.NewErrInvalidRequest("Prompt name is required", err))
			return
		}

		versionStr, err := pathParam(ctx, "version")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		var body prompt.UpdatePromptVersionLabelRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		updatedVersion, err := svc.Prompt.UpdatePromptVersionLabel(stdCtx, projectID, name, version, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update prompt version label", perrors.NewErrInternalServerError("Failed to update prompt version label", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt version label updated successfully", updatedVersion)
	})

	// Delete prompt version
	r.DELETE("/api/agent-server/prompts/{name}/versions/{version}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := pathParam(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Prompt name is required", perrors.NewErrInvalidRequest("Prompt name is required", err))
			return
		}

		versionStr, err := pathParam(ctx, "version")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		if err := svc.Prompt.DeletePromptVersion(stdCtx, projectID, name, version); err != nil {
			writeError(ctx, stdCtx, "Failed to delete prompt version", perrors.NewErrInternalServerError("Failed to delete prompt version", err))
			return
		}

		writeOK(ctx, stdCtx, "Prompt version deleted successfully", nil)
	})
}
