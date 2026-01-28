package controllers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

		// Create sandbox data directory structure for version 0
		if err := createSandboxDataDirectory(created.Name, created.Version); err != nil {
			// Log error but don't fail the request - sandbox data can be created later
			fmt.Printf("Warning: Failed to create sandbox data directory for agent %s: %v\n", created.Name, err)
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

		// Create sandbox data directory structure for the new version
		if err := createSandboxDataDirectory(created.Name, created.Version); err != nil {
			// Log error but don't fail the request - sandbox data can be created later
			fmt.Printf("Warning: Failed to create sandbox data directory for agent %s version %d: %v\n", created.Name, created.Version, err)
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

		// Create sandbox data directory structure for the new version
		if err := createSandboxDataDirectory(created.Name, created.Version); err != nil {
			// Log error but don't fail the request - sandbox data can be created later
			fmt.Printf("Warning: Failed to create sandbox data directory for agent %s version %d: %v\n", created.Name, created.Version, err)
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

	// Upload skills zip file to temp folder and parse SKILL.md
	r.POST("/api/agent-server/agent-configs/skills/upload", func(ctx *fasthttp.RequestCtx) {
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

		// Get agent config to verify it exists and get the version
		config, err := svc.AgentConfig.GetLatestByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Agent config not found", perrors.NewErrInvalidRequest("Agent config not found", err))
			return
		}

		// Parse multipart form
		multipartForm, err := ctx.MultipartForm()
		if err != nil {
			writeError(ctx, stdCtx, "Failed to parse multipart form", perrors.NewErrInvalidRequest("Failed to parse multipart form", err))
			return
		}
		defer multipartForm.RemoveAll()

		// Get the file from form
		fileHeaders := multipartForm.File["file"]
		if len(fileHeaders) == 0 {
			writeError(ctx, stdCtx, "No file provided", perrors.NewErrInvalidRequest("No file provided", nil))
			return
		}

		fileHeader := fileHeaders[0]
		if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".zip") {
			writeError(ctx, stdCtx, "File must be a .zip file", perrors.NewErrInvalidRequest("File must be a .zip file", nil))
			return
		}

		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			writeError(ctx, stdCtx, "Failed to open uploaded file", perrors.NewErrInternalServerError("Failed to open uploaded file", err))
			return
		}
		defer file.Close()

		// Get working directory
		wd, err := os.Getwd()
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get working directory", perrors.NewErrInternalServerError("Failed to get working directory", err))
			return
		}

		// Create temp directory path: sandbox-data/{AgentName}_{AgentVersion}/temp
		agentDirName := fmt.Sprintf("%s_%d", config.Name, config.Version)
		tempDir := filepath.Join(wd, "sandbox-data", agentDirName, "temp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			writeError(ctx, stdCtx, "Failed to create temp directory", perrors.NewErrInternalServerError("Failed to create temp directory", err))
			return
		}

		// Extract zip file name (without extension) to use as directory name
		zipName := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
		extractDir := filepath.Join(tempDir, zipName)

		// Remove existing temp directory if it exists
		if err := os.RemoveAll(extractDir); err != nil {
			writeError(ctx, stdCtx, "Failed to clean existing temp directory", perrors.NewErrInternalServerError("Failed to clean existing temp directory", err))
			return
		}

		// Create extract directory
		if err := os.MkdirAll(extractDir, 0755); err != nil {
			writeError(ctx, stdCtx, "Failed to create extract directory", perrors.NewErrInternalServerError("Failed to create extract directory", err))
			return
		}

		// Read zip file content
		zipData, err := io.ReadAll(file)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to read zip file", perrors.NewErrInternalServerError("Failed to read zip file", err))
			return
		}

		// Open zip reader using bytes.Reader which implements io.ReaderAt
		zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
		if err != nil {
			writeError(ctx, stdCtx, "Failed to open zip file", perrors.NewErrInvalidRequest("Failed to open zip file", err))
			return
		}

		// Extract all files from zip
		for _, zipFile := range zipReader.File {
			// Sanitize file path to prevent directory traversal
			// Clean the path and ensure it's within extractDir
			cleanName := filepath.Clean(zipFile.Name)
			if strings.HasPrefix(cleanName, "..") || strings.Contains(cleanName, "..") {
				writeError(ctx, stdCtx, "Invalid file path in zip", perrors.NewErrInvalidRequest("Invalid file path in zip", nil))
				return
			}
			filePath := filepath.Join(extractDir, cleanName)

			// Ensure the resolved path is still within extractDir
			absExtractDir, err := filepath.Abs(extractDir)
			if err != nil {
				writeError(ctx, stdCtx, "Failed to resolve extract directory", perrors.NewErrInternalServerError("Failed to resolve extract directory", err))
				return
			}
			absFilePath, err := filepath.Abs(filePath)
			if err != nil {
				writeError(ctx, stdCtx, "Failed to resolve file path", perrors.NewErrInternalServerError("Failed to resolve file path", err))
				return
			}
			if !strings.HasPrefix(absFilePath, absExtractDir) {
				writeError(ctx, stdCtx, "Invalid file path in zip", perrors.NewErrInvalidRequest("Invalid file path in zip", nil))
				return
			}

			// Create directory if needed
			if zipFile.FileInfo().IsDir() {
				if err := os.MkdirAll(filePath, zipFile.FileInfo().Mode()); err != nil {
					writeError(ctx, stdCtx, fmt.Sprintf("Failed to create directory: %s", zipFile.Name), perrors.NewErrInternalServerError("Failed to create directory", err))
					return
				}
				continue
			}

			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				writeError(ctx, stdCtx, fmt.Sprintf("Failed to create parent directory for: %s", zipFile.Name), perrors.NewErrInternalServerError("Failed to create parent directory", err))
				return
			}

			// Open file from zip
			rc, err := zipFile.Open()
			if err != nil {
				writeError(ctx, stdCtx, fmt.Sprintf("Failed to open file from zip: %s", zipFile.Name), perrors.NewErrInternalServerError("Failed to open file from zip", err))
				return
			}

			// Create destination file
			outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.FileInfo().Mode())
			if err != nil {
				rc.Close()
				writeError(ctx, stdCtx, fmt.Sprintf("Failed to create file: %s", zipFile.Name), perrors.NewErrInternalServerError("Failed to create file", err))
				return
			}

			// Copy file content
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()
			if err != nil {
				writeError(ctx, stdCtx, fmt.Sprintf("Failed to extract file: %s", zipFile.Name), perrors.NewErrInternalServerError("Failed to extract file", err))
				return
			}
		}

		// Parse SKILL.md to extract name and description
		skillMDPath := filepath.Join(extractDir, "SKILL.md")
		skillName, skillDescription, err := parseSkillMD(skillMDPath)
		if err != nil {
			// Clean up the extracted directory on error
			os.RemoveAll(extractDir)
			writeError(ctx, stdCtx, "Failed to parse SKILL.md: "+err.Error(), perrors.NewErrInvalidRequest("Failed to parse SKILL.md", err))
			return
		}

		// Build the relative temp path for response
		relTempPath := filepath.Join("sandbox-data", agentDirName, "temp", zipName)

		writeOK(ctx, stdCtx, "Skill file uploaded and extracted to temp successfully", agent_config.TempSkillUploadResponse{
			Name:        skillName,
			Description: skillDescription,
			TempPath:    relTempPath,
			SkillFolder: zipName,
		})
	})

	// Delete a skill from temp folder
	r.DELETE("/api/agent-server/agent-configs/skills/temp", func(ctx *fasthttp.RequestCtx) {
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

		skillFolder, err := requireStringQuery(ctx, "skill_folder")
		if err != nil {
			writeError(ctx, stdCtx, "Skill folder name is required", perrors.NewErrInvalidRequest("Skill folder name is required", err))
			return
		}

		// Get agent config to verify it exists and get the version
		config, err := svc.AgentConfig.GetLatestByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Agent config not found", perrors.NewErrInvalidRequest("Agent config not found", err))
			return
		}

		// Get working directory
		wd, err := os.Getwd()
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get working directory", perrors.NewErrInternalServerError("Failed to get working directory", err))
			return
		}

		// Sanitize skill folder name to prevent directory traversal
		cleanSkillFolder := filepath.Clean(skillFolder)
		if strings.Contains(cleanSkillFolder, "..") || strings.ContainsAny(cleanSkillFolder, "/\\") {
			writeError(ctx, stdCtx, "Invalid skill folder name", perrors.NewErrInvalidRequest("Invalid skill folder name", nil))
			return
		}

		// Build the temp skill path
		agentDirName := fmt.Sprintf("%s_%d", config.Name, config.Version)
		tempSkillPath := filepath.Join(wd, "sandbox-data", agentDirName, "temp", cleanSkillFolder)

		// Remove the temp skill directory
		if err := os.RemoveAll(tempSkillPath); err != nil {
			writeError(ctx, stdCtx, "Failed to delete temp skill", perrors.NewErrInternalServerError("Failed to delete temp skill", err))
			return
		}

		writeOK(ctx, stdCtx, "Temp skill deleted successfully", nil)
	})

	// Move skills from temp to actual location (called during save)
	r.POST("/api/agent-server/agent-configs/skills/commit", func(ctx *fasthttp.RequestCtx) {
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

		skillFolder, err := requireStringQuery(ctx, "skill_folder")
		if err != nil {
			writeError(ctx, stdCtx, "Skill folder name is required", perrors.NewErrInvalidRequest("Skill folder name is required", err))
			return
		}

		// Get agent config to verify it exists and get the version
		config, err := svc.AgentConfig.GetLatestByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Agent config not found", perrors.NewErrInvalidRequest("Agent config not found", err))
			return
		}

		// Get working directory
		wd, err := os.Getwd()
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get working directory", perrors.NewErrInternalServerError("Failed to get working directory", err))
			return
		}

		// Sanitize skill folder name to prevent directory traversal
		cleanSkillFolder := filepath.Clean(skillFolder)
		if strings.Contains(cleanSkillFolder, "..") || strings.ContainsAny(cleanSkillFolder, "/\\") {
			writeError(ctx, stdCtx, "Invalid skill folder name", perrors.NewErrInvalidRequest("Invalid skill folder name", nil))
			return
		}

		// Build paths
		agentDirName := fmt.Sprintf("%s_%d", config.Name, config.Version)
		tempSkillPath := filepath.Join(wd, "sandbox-data", agentDirName, "temp", cleanSkillFolder)
		skillsDir := filepath.Join(wd, "sandbox-data", agentDirName, "skills")
		destSkillPath := filepath.Join(skillsDir, cleanSkillFolder)

		// Check if temp skill exists
		if _, err := os.Stat(tempSkillPath); os.IsNotExist(err) {
			writeError(ctx, stdCtx, "Temp skill not found", perrors.NewErrInvalidRequest("Temp skill not found", nil))
			return
		}

		// Create skills directory if it doesn't exist
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			writeError(ctx, stdCtx, "Failed to create skills directory", perrors.NewErrInternalServerError("Failed to create skills directory", err))
			return
		}

		// Remove existing skill directory if it exists
		if err := os.RemoveAll(destSkillPath); err != nil {
			writeError(ctx, stdCtx, "Failed to clean existing skill directory", perrors.NewErrInternalServerError("Failed to clean existing skill directory", err))
			return
		}

		// Move from temp to skills directory
		if err := os.Rename(tempSkillPath, destSkillPath); err != nil {
			writeError(ctx, stdCtx, "Failed to move skill from temp", perrors.NewErrInternalServerError("Failed to move skill from temp", err))
			return
		}

		// Build the relative file location for SKILL.md
		fileLocation := filepath.Join("sandbox-data", agentDirName, "skills", cleanSkillFolder, "SKILL.md")

		writeOK(ctx, stdCtx, "Skill committed successfully", map[string]string{
			"file_location": fileLocation,
		})
	})

	// Delete a saved skill
	r.DELETE("/api/agent-server/agent-configs/skills", func(ctx *fasthttp.RequestCtx) {
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

		skillFolder, err := requireStringQuery(ctx, "skill_folder")
		if err != nil {
			writeError(ctx, stdCtx, "Skill folder name is required", perrors.NewErrInvalidRequest("Skill folder name is required", err))
			return
		}

		// Get agent config to verify it exists and get the version
		config, err := svc.AgentConfig.GetLatestByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Agent config not found", perrors.NewErrInvalidRequest("Agent config not found", err))
			return
		}

		// Get working directory
		wd, err := os.Getwd()
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get working directory", perrors.NewErrInternalServerError("Failed to get working directory", err))
			return
		}

		// Sanitize skill folder name to prevent directory traversal
		cleanSkillFolder := filepath.Clean(skillFolder)
		if strings.Contains(cleanSkillFolder, "..") || strings.ContainsAny(cleanSkillFolder, "/\\") {
			writeError(ctx, stdCtx, "Invalid skill folder name", perrors.NewErrInvalidRequest("Invalid skill folder name", nil))
			return
		}

		// Build the skill path
		agentDirName := fmt.Sprintf("%s_%d", config.Name, config.Version)
		skillPath := filepath.Join(wd, "sandbox-data", agentDirName, "skills", cleanSkillFolder)

		// Remove the skill directory
		if err := os.RemoveAll(skillPath); err != nil {
			writeError(ctx, stdCtx, "Failed to delete skill", perrors.NewErrInternalServerError("Failed to delete skill", err))
			return
		}

		writeOK(ctx, stdCtx, "Skill deleted successfully", nil)
	})
}

// createSandboxDataDirectory creates the sandbox data directory structure for an agent
// If version > 0, it copies the workspace content from version 0
func createSandboxDataDirectory(agentName string, version int) error {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create agent directory name: {AgentName}_{Version}
	agentDirName := fmt.Sprintf("%s_%d", agentName, version)
	agentDir := filepath.Join(wd, "sandbox-data", agentDirName)

	// Create skills directory under workspace
	skillsDir := filepath.Join(agentDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// If this is a new version (version > 0), copy workspace and namespaces content from version 0
	if version > 0 {
		version0DirName := fmt.Sprintf("%s_0", agentName)
		version0Dir := filepath.Join(wd, "sandbox-data", version0DirName)

		// Copy skills directory from version 0
		version0SkillsDir := filepath.Join(version0Dir, "skills")
		if _, err := os.Stat(version0SkillsDir); err == nil {
			// Copy skills content from version 0 to new version
			if err := copyDirectory(version0SkillsDir, skillsDir); err != nil {
				return fmt.Errorf("failed to copy workspace from version 0: %w", err)
			}
		}
	}

	return nil
}

// copyDirectory recursively copies a directory from source to destination
func copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory with same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy file content
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// parseSkillMD parses the SKILL.md file and extracts name and description from YAML frontmatter
func parseSkillMD(path string) (name string, description string, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("SKILL.md not found in zip")
	}

	contentStr := string(content)

	// Check if the file starts with YAML frontmatter delimiter
	if !strings.HasPrefix(contentStr, "---") {
		return "", "", fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	// Find the end of the frontmatter
	endIdx := strings.Index(contentStr[3:], "---")
	if endIdx == -1 {
		return "", "", fmt.Errorf("SKILL.md has invalid YAML frontmatter (missing closing ---)")
	}

	frontmatter := contentStr[3 : endIdx+3]

	// Parse YAML frontmatter line by line
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}

	if name == "" {
		return "", "", fmt.Errorf("SKILL.md frontmatter must contain 'name' field")
	}
	if description == "" {
		return "", "", fmt.Errorf("SKILL.md frontmatter must contain 'description' field")
	}

	return name, description, nil
}
