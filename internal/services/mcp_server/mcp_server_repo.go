package mcp_server

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MCPServerRepo handles database operations for MCP servers
type MCPServerRepo struct {
	db *sqlx.DB
}

// NewMCPServerRepo creates a new MCP server repository
func NewMCPServerRepo(db *sqlx.DB) *MCPServerRepo {
	return &MCPServerRepo{db: db}
}

// Create creates a new MCP server
func (r *MCPServerRepo) Create(ctx context.Context, projectId uuid.UUID, req *CreateMCPServerRequest) (*MCPServer, error) {
	query := `
		INSERT INTO mcp_servers (name, endpoint, headers, project_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, endpoint, headers, created_at, updated_at
	`

	var server MCPServer
	err := r.db.GetContext(ctx, &server, query, req.Name, req.Endpoint, req.Headers, projectId)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return &server, nil
}

// GetByID retrieves an MCP server by ID
func (r *MCPServerRepo) GetByID(ctx context.Context, projectId uuid.UUID, id uuid.UUID) (*MCPServer, error) {
	query := `
		SELECT id, name, endpoint, headers, created_at, updated_at
		FROM mcp_servers
		WHERE id = $1 and project_id = $2
	`

	var server MCPServer
	err := r.db.GetContext(ctx, &server, query, id, projectId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("MCP server not found")
		}
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	return &server, nil
}

// GetByName retrieves an MCP server by name
func (r *MCPServerRepo) GetByName(ctx context.Context, projectId uuid.UUID, name string) (*MCPServer, error) {
	query := `
		SELECT id, name, endpoint, headers, created_at, updated_at
		FROM mcp_servers
		WHERE name = $1 and project_id = $2
	`

	var server MCPServer
	err := r.db.GetContext(ctx, &server, query, name, projectId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("MCP server not found")
		}
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	return &server, nil
}

// GetByIDs retrieves multiple MCP servers by IDs in a single query
func (r *MCPServerRepo) GetByIDs(ctx context.Context, projectId uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*MCPServer, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*MCPServer), nil
	}

	query, args, err := sqlx.In(`
		SELECT id, name, endpoint, headers, created_at, updated_at
		FROM mcp_servers
		WHERE id IN (?) AND project_id = ?
	`, ids, projectId)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	query = r.db.Rebind(query)
	var servers []*MCPServer
	err = r.db.SelectContext(ctx, &servers, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP servers: %w", err)
	}

	// Convert to map for easy lookup
	result := make(map[uuid.UUID]*MCPServer, len(servers))
	for _, server := range servers {
		result[server.ID] = server
	}

	return result, nil
}

// List retrieves all MCP servers
func (r *MCPServerRepo) List(ctx context.Context, projectId uuid.UUID) ([]*MCPServer, error) {
	query := `
		SELECT id, name, endpoint, headers, project_id, created_at, updated_at
		FROM mcp_servers
		WHERE project_id = $1
		ORDER BY created_at DESC
	`

	var servers []*MCPServer
	err := r.db.SelectContext(ctx, &servers, query, projectId)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}

	return servers, nil
}

// Update updates an MCP server
func (r *MCPServerRepo) Update(ctx context.Context, projectId uuid.UUID, id uuid.UUID, req *UpdateMCPServerRequest) (*MCPServer, error) {
	// Build dynamic query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Endpoint != nil {
		setParts = append(setParts, fmt.Sprintf("endpoint = $%d", argIndex))
		args = append(args, *req.Endpoint)
		argIndex++
	}

	if req.Headers != nil {
		setParts = append(setParts, fmt.Sprintf("headers = $%d", argIndex))
		args = append(args, *req.Headers)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, projectId, id)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id.String())
	args = append(args, projectId.String())
	argIndex += 1

	// Join all set parts with commas
	setClause := ""
	for i, part := range setParts {
		if i > 0 {
			setClause += ", "
		}
		setClause += part
	}

	query := fmt.Sprintf(`
		UPDATE mcp_servers
		SET %s
		WHERE id = $%d and project_id = $%d
		RETURNING id, name, endpoint, headers, created_at, updated_at
	`, setClause, argIndex-1, argIndex)

	var server MCPServer
	err := r.db.GetContext(ctx, &server, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("MCP server not found")
		}
		return nil, fmt.Errorf("failed to update MCP server: %w", err)
	}

	return &server, nil
}

// Delete deletes an MCP server
func (r *MCPServerRepo) Delete(ctx context.Context, projectId uuid.UUID, id uuid.UUID) error {
	query := `DELETE FROM mcp_servers WHERE id = $1 and project_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, projectId)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("MCP server not found")
	}

	return nil
}
