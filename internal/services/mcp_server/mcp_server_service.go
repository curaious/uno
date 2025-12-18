package mcp_server

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

// MCPServerService handles business logic for MCP servers
type MCPServerService struct {
	repo *MCPServerRepo
}

// NewMCPServerService creates a new MCP server service
func NewMCPServerService(repo *MCPServerRepo) *MCPServerService {
	return &MCPServerService{repo: repo}
}

// Create creates a new MCP server
func (s *MCPServerService) Create(ctx context.Context, projectID uuid.UUID, req *CreateMCPServerRequest) (*MCPServer, error) {
	// Validate endpoint URL
	if _, err := url.Parse(req.Endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Check if server with same name already exists
	existing, err := s.repo.GetByName(ctx, projectID, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("MCP server with name '%s' already exists", req.Name)
	}

	// Ensure headers is not nil
	if req.Headers == nil {
		req.Headers = make(HeadersMap)
	}

	// Create the server
	server, err := s.repo.Create(ctx, projectID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return server, nil
}

// GetByID retrieves an MCP server by ID
func (s *MCPServerService) GetByID(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (*MCPServer, error) {
	server, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	return server, nil
}

// GetByName retrieves an MCP server by name
func (s *MCPServerService) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*MCPServer, error) {
	server, err := s.repo.GetByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	return server, nil
}

// GetByIDs retrieves multiple MCP servers by IDs in a single query
func (s *MCPServerService) GetByIDs(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*MCPServer, error) {
	servers, err := s.repo.GetByIDs(ctx, projectID, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP servers: %w", err)
	}

	return servers, nil
}

// List retrieves all MCP servers
func (s *MCPServerService) List(ctx context.Context, projectID uuid.UUID) ([]*MCPServer, error) {
	servers, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}

	return servers, nil
}

// Update updates an MCP server
func (s *MCPServerService) Update(ctx context.Context, projectID uuid.UUID, id uuid.UUID, req *UpdateMCPServerRequest) (*MCPServer, error) {
	// Check if server exists
	existing, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing MCP server: %w", err)
	}

	// If updating name, check if new name already exists
	if req.Name != nil && *req.Name != existing.Name {
		existingByName, err := s.repo.GetByName(ctx, projectID, *req.Name)
		if err == nil && existingByName != nil {
			return nil, fmt.Errorf("MCP server with name '%s' already exists", *req.Name)
		}
	}

	// If updating endpoint, validate URL
	if req.Endpoint != nil {
		if _, err := url.Parse(*req.Endpoint); err != nil {
			return nil, fmt.Errorf("invalid endpoint URL: %w", err)
		}
	}

	// Update the server
	server, err := s.repo.Update(ctx, projectID, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update MCP server: %w", err)
	}

	return server, nil
}

// Delete deletes an MCP server
func (s *MCPServerService) Delete(ctx context.Context, projectID uuid.UUID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, projectID, id)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server: %w", err)
	}

	return nil
}
