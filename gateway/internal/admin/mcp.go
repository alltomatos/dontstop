package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MCPTool is one exposed MCP tool.
type MCPTool struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Scope   string `json:"scope"`
	Enabled bool   `json:"enabled"`
}

// MCPStatus is the console MCP snapshot.
type MCPStatus struct {
	Enabled   bool      `json:"enabled"`
	Transport string    `json:"transport"`
	Tools     []MCPTool `json:"tools"`
}

var mcpTools = []MCPTool{
	{ID: "dontstop_chat", Name: "dontstop_chat", Scope: "chat:write"},
	{ID: "dontstop_models", Name: "dontstop_models", Scope: "models:read"},
	{ID: "dontstop_embeddings", Name: "dontstop_embeddings", Scope: "embeddings:write"},
	{ID: "dontstop_web_search", Name: "dontstop_web_search", Scope: "web:read"},
	{ID: "dontstop_usage", Name: "dontstop_usage", Scope: "usage:read"},
	{ID: "dontstop_health", Name: "dontstop_health", Scope: "models:read"},
}

// MCPService persists MCP settings.
type MCPService struct {
	pool *pgxpool.Pool
}

// NewMCPService creates an MCP service.
func NewMCPService(pool *pgxpool.Pool) *MCPService {
	return &MCPService{pool: pool}
}

// Status returns the current settings and tools.
func (s *MCPService) Status(ctx context.Context, tenantID string) (MCPStatus, error) {
	status := MCPStatus{Enabled: false, Transport: "stdio", Tools: make([]MCPTool, 0, len(mcpTools))}
	err := s.pool.QueryRow(ctx, `SELECT enabled, transport FROM mcp_settings WHERE tenant_id = $1`, tenantID).Scan(&status.Enabled, &status.Transport)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return MCPStatus{}, fmt.Errorf("query MCP settings: %w", err)
	}
	overrides := map[string]bool{}
	rows, err := s.pool.Query(ctx, `SELECT tool_id, enabled FROM mcp_tool_settings WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return MCPStatus{}, fmt.Errorf("query MCP tools: %w", err)
	}
	for rows.Next() {
		var toolID string
		var enabled bool
		if err := rows.Scan(&toolID, &enabled); err != nil {
			rows.Close()
			return MCPStatus{}, fmt.Errorf("scan MCP tool: %w", err)
		}
		overrides[toolID] = enabled
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return MCPStatus{}, fmt.Errorf("iterate MCP tools: %w", err)
	}
	for _, tool := range mcpTools {
		if enabled, ok := overrides[tool.ID]; ok {
			tool.Enabled = enabled
		} else {
			tool.Enabled = true
		}
		status.Tools = append(status.Tools, tool)
	}
	return status, nil
}

// UpdateSettings updates enabled and/or transport.
func (s *MCPService) UpdateSettings(ctx context.Context, tenantID string, enabled *bool, transport *string) (MCPStatus, error) {
	if transport != nil {
		value := strings.TrimSpace(*transport)
		switch value {
		case "stdio", "sse", "streamable-http":
			*transport = value
		default:
			return MCPStatus{}, fmt.Errorf("%w: invalid MCP transport", ErrInvalidInput)
		}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO mcp_settings (tenant_id, enabled, transport)
		VALUES ($1, COALESCE($2::boolean, false), COALESCE($3::text, 'stdio'))
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = COALESCE($2::boolean, mcp_settings.enabled),
			transport = COALESCE($3::text, mcp_settings.transport),
			updated_at = now()
	`, tenantID, enabled, transport)
	if err != nil {
		return MCPStatus{}, fmt.Errorf("update MCP settings: %w", err)
	}
	return s.Status(ctx, tenantID)
}

// UpdateTool enables or disables one tool.
func (s *MCPService) UpdateTool(ctx context.Context, tenantID, toolID string, enabled bool) (MCPStatus, error) {
	valid := false
	for _, tool := range mcpTools {
		if tool.ID == toolID {
			valid = true
			break
		}
	}
	if !valid {
		return MCPStatus{}, ErrNotFound
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO mcp_tool_settings (tenant_id, tool_id, enabled)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, tool_id) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now()
	`, tenantID, toolID, enabled)
	if err != nil {
		return MCPStatus{}, fmt.Errorf("update MCP tool: %w", err)
	}
	return s.Status(ctx, tenantID)
}
