package controllers

import (
	"strconv"

	"github.com/curaious/uno/internal/perrors"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

func RegisterAgentConfigRoutes(r *router.Router, svc *services.Services) {
	// Create agent config
	r.POST("/api/agent-server/agent-configs", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body agent_config.CreateAgentConfigRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", nil))
			return
		}

		created, err := svc.AgentConfig.Create(stdCtx, projectID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create agent config", perrors.NewErrInternalServerError("Failed to create agent config", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config created successfully", created)
	})

	// List agent configs
	r.GET("/api/agent-server/agent-configs", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		configs, err := svc.AgentConfig.List(stdCtx, projectID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list agent configs", perrors.NewErrInternalServerError("Failed to list agent configs", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent configs retrieved successfully", configs)
	})

	// Get agent config by ID
	r.GET("/api/agent-server/agent-configs/{id}", func(ctx *fasthttp.RequestCtx) {
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

		config, err := svc.AgentConfig.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config retrieved successfully", config)
	})

	// Get agent config by name (latest version)
	r.GET("/api/agent-server/agent-configs/by-name", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Agent config name is required", perrors.NewErrInvalidRequest("Agent config name is required", err))
			return
		}

		// Check if version is provided
		versionStr := string(ctx.QueryArgs().Peek("version"))
		if versionStr != "" {
			version, err := strconv.Atoi(versionStr)
			if err != nil {
				writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
				return
			}

			config, err := svc.AgentConfig.GetByNameAndVersion(stdCtx, projectID, name, version)
			if err != nil {
				writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
				return
			}

			writeOK(ctx, stdCtx, "Agent config retrieved successfully", config)
			return
		}

		// Get latest version
		config, err := svc.AgentConfig.GetLatestByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config retrieved successfully", config)
	})

	// List all versions of an agent config
	r.GET("/api/agent-server/agent-configs/by-name/versions", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Agent config name is required", perrors.NewErrInvalidRequest("Agent config name is required", err))
			return
		}

		configs, err := svc.AgentConfig.ListVersions(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list agent config versions", perrors.NewErrInternalServerError("Failed to list agent config versions", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config versions retrieved successfully", configs)
	})

	// Create new version of agent config
	r.POST("/api/agent-server/agent-configs/by-name/versions", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Agent config name is required", perrors.NewErrInvalidRequest("Agent config name is required", err))
			return
		}

		var body agent_config.UpdateAgentConfigRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		created, err := svc.AgentConfig.CreateVersion(stdCtx, projectID, name, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create agent config version", perrors.NewErrInternalServerError("Failed to create agent config version", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config version created successfully", created)
	})

	// Delete all versions of an agent config
	r.DELETE("/api/agent-server/agent-configs/by-name", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Agent config name is required", perrors.NewErrInvalidRequest("Agent config name is required", err))
			return
		}

		if err := svc.AgentConfig.Delete(stdCtx, projectID, name); err != nil {
			writeError(ctx, stdCtx, "Failed to delete agent config", perrors.NewErrInternalServerError("Failed to delete agent config", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config deleted successfully", nil)
	})

	// Delete a specific version of an agent config
	r.DELETE("/api/agent-server/agent-configs/by-name/versions", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "Agent config name is required", perrors.NewErrInvalidRequest("Agent config name is required", err))
			return
		}

		versionStr := string(ctx.QueryArgs().Peek("version"))
		if versionStr == "" {
			writeError(ctx, stdCtx, "Version is required", perrors.NewErrInvalidRequest("Version is required", nil))
			return
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid version format", perrors.NewErrInvalidRequest("Invalid version format", err))
			return
		}

		if err := svc.AgentConfig.DeleteVersion(stdCtx, projectID, name, version); err != nil {
			writeError(ctx, stdCtx, "Failed to delete agent config version", perrors.NewErrInternalServerError("Failed to delete agent config version", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config version deleted successfully", nil)
	})
}
