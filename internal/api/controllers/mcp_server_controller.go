package controllers

import (
	"errors"

	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/mcp_server"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/valyala/fasthttp"

	"github.com/curaious/uno/internal/perrors"
)

func RegisterMCPServerRoutes(r *router.Router, svc *services.Services) {
	// Create MCP server
	r.POST("/api/agent-server/mcp-servers", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		var body mcp_server.CreateMCPServerRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if body.Endpoint == "" {
			writeError(ctx, stdCtx, "Endpoint is required", perrors.NewErrInvalidRequest("Endpoint is required", errors.New("endpoint is required")))
			return
		}

		// Ensure headers is not nil
		if body.Headers == nil {
			body.Headers = make(mcp_server.HeadersMap)
		}

		server, err := svc.MCPServer.Create(stdCtx, projectID, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create MCP server", perrors.NewErrInternalServerError("Failed to create MCP server", err))
			return
		}

		writeOK(ctx, stdCtx, "MCP server created successfully", server)
	})

	// List MCP servers
	r.GET("/api/agent-server/mcp-servers", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		servers, err := svc.MCPServer.List(stdCtx, projectID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list MCP servers", perrors.NewErrInternalServerError("Failed to list MCP servers", err))
			return
		}

		writeOK(ctx, stdCtx, "MCP servers retrieved successfully", servers)
	})

	// Get MCP server by ID
	r.GET("/api/agent-server/mcp-servers/{id}", func(ctx *fasthttp.RequestCtx) {
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

		server, err := svc.MCPServer.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get MCP server", perrors.NewErrInternalServerError("Failed to get MCP server", err))
			return
		}

		writeOK(ctx, stdCtx, "MCP server retrieved successfully", server)
	})

	// Get MCP server by name
	r.GET("/api/agent-server/mcp-servers/by-name", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		projectID, err := requireUUIDQuery(ctx, "project_id")
		if err != nil {
			writeError(ctx, stdCtx, "Project ID is required", perrors.NewErrInvalidRequest("Project ID is required", err))
			return
		}

		name, err := requireStringQuery(ctx, "name")
		if err != nil {
			writeError(ctx, stdCtx, "MCP Server name is required", perrors.NewErrInvalidRequest("MCP Server name is required", err))
			return
		}

		server, err := svc.MCPServer.GetByName(stdCtx, projectID, name)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get MCP server", perrors.NewErrInternalServerError("Failed to get MCP server", err))
			return
		}

		writeOK(ctx, stdCtx, "MCP server retrieved successfully", server)
	})

	// Inspect MCP server
	r.GET("/api/agent-server/mcp-servers/{id}/inspect", func(ctx *fasthttp.RequestCtx) {
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

		server, err := svc.MCPServer.GetByID(stdCtx, projectID, id)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get MCP server", perrors.NewErrInternalServerError("Failed to get MCP server", err))
			return
		}

		srv, err := mcpclient.NewSSEClient(stdCtx, server.Endpoint, mcpclient.WithHeaders(server.Headers))
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create MCP client", perrors.NewErrInternalServerError("Failed to create MCP client", errors.New("could not create MCP client")))
			return
		}

		cli, err := srv.GetClient(ctx, nil)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to initialize MCP Client", perrors.NewErrInternalServerError("Failed to initialize MCP client", err))
			return
		}

		res := InspectMCPResponse{
			Tools: cli.Tools,
		}

		writeOK(ctx, stdCtx, "MCP server retrieved successfully", res)
	})

	// Update MCP server
	r.PUT("/api/agent-server/mcp-servers/{id}", func(ctx *fasthttp.RequestCtx) {
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

		var body mcp_server.UpdateMCPServerRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name != nil && *body.Name == "" {
			writeError(ctx, stdCtx, "Name cannot be empty", perrors.NewErrInvalidRequest("Name cannot be empty", errors.New("name cannot be empty")))
			return
		}

		if body.Endpoint != nil && *body.Endpoint == "" {
			writeError(ctx, stdCtx, "Endpoint cannot be empty", perrors.NewErrInvalidRequest("Endpoint cannot be empty", errors.New("endpoint cannot be empty")))
			return
		}

		server, err := svc.MCPServer.Update(stdCtx, projectID, id, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update MCP server", perrors.NewErrInternalServerError("Failed to update MCP server", err))
			return
		}

		writeOK(ctx, stdCtx, "MCP server updated successfully", server)
	})

	// Delete MCP server
	r.DELETE("/api/agent-server/mcp-servers/{id}", func(ctx *fasthttp.RequestCtx) {
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

		if err := svc.MCPServer.Delete(stdCtx, projectID, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete MCP server", perrors.NewErrInternalServerError("Failed to delete MCP server", err))
			return
		}

		writeOK(ctx, stdCtx, "MCP server deleted successfully", nil)
	})
}

type InspectMCPResponse struct {
	Tools []mcp.Tool `json:"tools"`
}
