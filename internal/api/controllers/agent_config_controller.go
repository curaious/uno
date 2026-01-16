package controllers

import (
	"encoding/json"
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

	// List all versions of an agent config by agent_id
	r.GET("/api/agent-server/agent-configs/{id}/versions", func(ctx *fasthttp.RequestCtx) {
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

		// Get config to get agent_id
		configs, err := svc.AgentConfig.ListVersions(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list agent config versions", perrors.NewErrInternalServerError("Failed to list agent config versions", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config versions retrieved successfully", configs)
	})

	// List all versions of an agent config by name (for backward compatibility)
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

		configs, err := svc.AgentConfig.ListVersionsByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list agent config versions", perrors.NewErrInternalServerError("Failed to list agent config versions", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config versions retrieved successfully", configs)
	})

	// Update version 0 (mutable) of agent config by ID
	r.POST("/api/agent-server/agent-configs/{id}/versions", func(ctx *fasthttp.RequestCtx) {
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

		// Get config to get agent_id
		config, err := svc.AgentConfig.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
			return
		}

		var body agent_config.UpdateAgentConfigRequest
		if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		updated, err := svc.AgentConfig.UpdateVersion0(stdCtx, config.AgentID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update agent config", perrors.NewErrInternalServerError("Failed to update agent config", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config updated successfully", updated)
	})

	// Update version 0 (mutable) of agent config by name (for backward compatibility)
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

		updated, err := svc.AgentConfig.UpdateVersion0ByName(stdCtx, projectID, name, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update agent config", perrors.NewErrInternalServerError("Failed to update agent config", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config updated successfully", updated)
	})

	// Create new immutable version from version 0 by ID
	r.POST("/api/agent-server/agent-configs/{id}/versions/create", func(ctx *fasthttp.RequestCtx) {
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

		// Get config to get agent_id
		config, err := svc.AgentConfig.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
			return
		}

		created, err := svc.AgentConfig.CreateVersion(stdCtx, config.AgentID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create agent config version", perrors.NewErrInternalServerError("Failed to create agent config version", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config version created successfully", created)
	})

	// Create new immutable version from version 0 by name (for backward compatibility)
	r.POST("/api/agent-server/agent-configs/by-name/versions/create", func(ctx *fasthttp.RequestCtx) {
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

		created, err := svc.AgentConfig.CreateVersionByName(stdCtx, projectID, name)
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

		if err := svc.AgentConfig.DeleteByName(stdCtx, projectID, name); err != nil {
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

		// Get agent_id first
		config, err := svc.AgentConfig.GetLatestByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
			return
		}

		if err := svc.AgentConfig.DeleteVersion(stdCtx, config.AgentID, version); err != nil {
			writeError(ctx, stdCtx, "Failed to delete agent config version", perrors.NewErrInternalServerError("Failed to delete agent config version", err))
			return
		}

		writeOK(ctx, stdCtx, "Agent config version deleted successfully", nil)
	})

	// Create alias by agent config ID
	r.POST("/api/agent-server/agent-configs/{id}/aliases", func(ctx *fasthttp.RequestCtx) {
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

		// Get config to get agent_id
		config, err := svc.AgentConfig.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get agent config", perrors.NewErrInternalServerError("Failed to get agent config", err))
			return
		}

		var req agent_config.CreateAliasRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		alias, err := svc.AgentConfig.CreateAliasByAgentID(stdCtx, projectID, config.AgentID, &req)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create alias", perrors.NewErrInternalServerError("Failed to create alias", err))
			return
		}

		writeOK(ctx, stdCtx, "Alias created successfully", alias)
	})

	// Create alias by name (for backward compatibility)
	r.POST("/api/agent-server/agent-configs/by-name/aliases", func(ctx *fasthttp.RequestCtx) {
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

		var req agent_config.CreateAliasRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		alias, err := svc.AgentConfig.CreateAlias(stdCtx, projectID, name, &req)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create alias", perrors.NewErrInternalServerError("Failed to create alias", err))
			return
		}

		writeOK(ctx, stdCtx, "Alias created successfully", alias)
	})

	// List aliases by agent config ID
	r.GET("/api/agent-server/agent-configs/{id}/aliases", func(ctx *fasthttp.RequestCtx) {
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

		aliases, err := svc.AgentConfig.ListAliasesByAgentID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list aliases", perrors.NewErrInternalServerError("Failed to list aliases", err))
			return
		}

		writeOK(ctx, stdCtx, "Aliases retrieved successfully", aliases)
	})

	// List aliases by name (for backward compatibility)
	r.GET("/api/agent-server/agent-configs/by-name/aliases", func(ctx *fasthttp.RequestCtx) {
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

		aliases, err := svc.AgentConfig.ListAliases(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list aliases", perrors.NewErrInternalServerError("Failed to list aliases", err))
			return
		}

		writeOK(ctx, stdCtx, "Aliases retrieved successfully", aliases)
	})

	// Get alias by ID
	r.GET("/api/agent-server/agent-configs/aliases/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		id, err := uuid.Parse(string(ctx.UserValue("id").(string)))
		if err != nil {
			writeError(ctx, stdCtx, "Invalid alias ID", perrors.NewErrInvalidRequest("Invalid alias ID", err))
			return
		}

		alias, err := svc.AgentConfig.GetAlias(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get alias", perrors.NewErrInternalServerError("Failed to get alias", err))
			return
		}

		writeOK(ctx, stdCtx, "Alias retrieved successfully", alias)
	})

	// Update alias
	r.PUT("/api/agent-server/agent-configs/aliases/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		id, err := uuid.Parse(string(ctx.UserValue("id").(string)))
		if err != nil {
			writeError(ctx, stdCtx, "Invalid alias ID", perrors.NewErrInvalidRequest("Invalid alias ID", err))
			return
		}

		var req agent_config.UpdateAliasRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		alias, err := svc.AgentConfig.UpdateAlias(stdCtx, projectID, id, &req)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update alias", perrors.NewErrInternalServerError("Failed to update alias", err))
			return
		}

		writeOK(ctx, stdCtx, "Alias updated successfully", alias)
	})

	// Delete alias
	r.DELETE("/api/agent-server/agent-configs/aliases/:id", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		id, err := uuid.Parse(string(ctx.UserValue("id").(string)))
		if err != nil {
			writeError(ctx, stdCtx, "Invalid alias ID", perrors.NewErrInvalidRequest("Invalid alias ID", err))
			return
		}

		if err := svc.AgentConfig.DeleteAlias(stdCtx, projectID, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete alias", perrors.NewErrInternalServerError("Failed to delete alias", err))
			return
		}

		writeOK(ctx, stdCtx, "Alias deleted successfully", nil)
	})
}
