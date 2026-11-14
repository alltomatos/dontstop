package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/mcp/catalog"
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

// MCPConfig is the tenant-level MCP runtime configuration.
type MCPConfig struct {
	Enabled   bool   `json:"enabled"`
	Transport string `json:"transport"`
}

// MCPActivity is the 24h runtime activity summary.
type MCPActivity struct {
	TotalCalls   int64          `json:"total_calls"`
	SuccessRate  float64        `json:"success_rate"`
	AvgLatencyMS float64        `json:"avg_latency_ms"`
	TopTools     []MCPToolCount `json:"top_tools"`
	Hours        int            `json:"hours"`
	GeneratedAt  time.Time      `json:"generated_at"`
}

// MCPToolCount is one entry in the activity ranking.
type MCPToolCount struct {
	Tool  string `json:"tool"`
	Count int64  `json:"count"`
}

var mcpTools = buildMCPTools()

func buildMCPTools() []MCPTool {
	tools := make([]MCPTool, 0, len(catalog.Tools))
	for _, tool := range catalog.Tools {
		tools = append(tools, MCPTool{ID: tool.ID, Name: tool.Name, Scope: tool.Scope})
	}
	return tools
}

// MCPService persists MCP settings and reads runtime activity.
type MCPService struct {
	pool *pgxpool.Pool
}

// NewMCPService creates an MCP service.
func NewMCPService(pool *pgxpool.Pool) *MCPService {
	return &MCPService{pool: pool}
}

// Settings returns the tenant MCP configuration, defaulting to disabled/stdio.
func (s *MCPService) Settings(ctx context.Context, tenantID string) (MCPConfig, error) {
	config := MCPConfig{Enabled: false, Transport: "stdio"}
	err := s.pool.QueryRow(ctx, `SELECT enabled, transport FROM mcp_settings WHERE tenant_id = $1`, tenantID).Scan(&config.Enabled, &config.Transport)
	if errors.Is(err, pgx.ErrNoRows) {
		return config, nil
	}
	if err != nil {
		return MCPConfig{}, fmt.Errorf("query MCP settings: %w", err)
	}
	return config, nil
}

// ToolOverrides returns the per-tool enabled overrides for a tenant.
func (s *MCPService) ToolOverrides(ctx context.Context, tenantID string) (map[string]bool, error) {
	overrides := map[string]bool{}
	rows, err := s.pool.Query(ctx, `SELECT tool_id, enabled FROM mcp_tool_settings WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query MCP tools: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var toolID string
		var enabled bool
		if err := rows.Scan(&toolID, &enabled); err != nil {
			return nil, fmt.Errorf("scan MCP tool: %w", err)
		}
		overrides[toolID] = enabled
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MCP tools: %w", err)
	}
	return overrides, nil
}

// Status returns the current settings and tools.
func (s *MCPService) Status(ctx context.Context, tenantID string) (MCPStatus, error) {
	config, err := s.Settings(ctx, tenantID)
	if err != nil {
		return MCPStatus{}, err
	}
	overrides, err := s.ToolOverrides(ctx, tenantID)
	if err != nil {
		return MCPStatus{}, err
	}
	status := MCPStatus{
		Enabled:   config.Enabled,
		Transport: config.Transport,
		Tools:     make([]MCPTool, 0, len(mcpTools)),
	}
	for _, tool := range mcpTools {
		tool.Enabled = true
		if enabled, ok := overrides[tool.ID]; ok {
			tool.Enabled = enabled
		}
		status.Tools = append(status.Tools, tool)
	}
	return status, nil
}

// UpdateSettings updates enabled and/or transport and records the change.
func (s *MCPService) UpdateSettings(ctx context.Context, tenantID, actor string, enabled *bool, transport *string) (MCPStatus, error) {
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
	if enabled != nil {
		event := "mcp.disable"
		if *enabled {
			event = "mcp.enable"
		}
		s.audit(ctx, tenantID, actor, event, map[string]any{"enabled": *enabled})
	}
	if transport != nil {
		s.audit(ctx, tenantID, actor, "mcp.transport", map[string]any{"transport": *transport})
	}
	return s.Status(ctx, tenantID)
}

// UpdateTool enables or disables one tool and records the change.
func (s *MCPService) UpdateTool(ctx context.Context, tenantID, actor, toolID string, enabled bool) (MCPStatus, error) {
	if _, ok := catalog.Find(toolID); !ok {
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
	event := "mcp.tool.disable"
	if enabled {
		event = "mcp.tool.enable"
	}
	s.audit(ctx, tenantID, actor, event, map[string]any{"tool": toolID, "enabled": enabled})
	return s.Status(ctx, tenantID)
}

// Restart records an audited soft restart and returns the current status.
func (s *MCPService) Restart(ctx context.Context, tenantID, actor string) (MCPStatus, error) {
	s.audit(ctx, tenantID, actor, "mcp.restart", nil)
	return s.Status(ctx, tenantID)
}

// Activity returns the runtime tool-call aggregates for the last hours.
func (s *MCPService) Activity(ctx context.Context, tenantID string, hours int) (MCPActivity, error) {
	if hours <= 0 || hours > 168 {
		hours = 24
	}
	activity := MCPActivity{Hours: hours, TopTools: []MCPToolCount{}, GeneratedAt: time.Now().UTC()}
	var success int64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*)::bigint,
		       count(*) FILTER (WHERE payload->>'status' = 'ok')::bigint,
		       COALESCE(avg(NULLIF(payload->>'duration_ms', '')::float8), 0)::float8
		FROM audit_outbox
		WHERE event_type = 'mcp.tool.call'
		  AND payload->>'tenant_id' = $1
		  AND created_at > now() - make_interval(hours => $2)
	`, tenantID, hours).Scan(&activity.TotalCalls, &success, &activity.AvgLatencyMS); err != nil {
		return MCPActivity{}, fmt.Errorf("query MCP activity: %w", err)
	}
	if activity.TotalCalls > 0 {
		activity.SuccessRate = float64(success) / float64(activity.TotalCalls) * 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT payload->>'tool', count(*)::bigint
		FROM audit_outbox
		WHERE event_type = 'mcp.tool.call'
		  AND payload->>'tenant_id' = $1
		  AND payload ? 'tool'
		  AND created_at > now() - make_interval(hours => $2)
		GROUP BY 1
		ORDER BY 2 DESC, 1
		LIMIT 5
	`, tenantID, hours)
	if err != nil {
		return MCPActivity{}, fmt.Errorf("query MCP top tools: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item MCPToolCount
		if err := rows.Scan(&item.Tool, &item.Count); err != nil {
			return MCPActivity{}, fmt.Errorf("scan MCP top tool: %w", err)
		}
		activity.TopTools = append(activity.TopTools, item)
	}
	if err := rows.Err(); err != nil {
		return MCPActivity{}, fmt.Errorf("iterate MCP top tools: %w", err)
	}
	return activity, nil
}

// RecordAudit writes an MCP audit event without payload/PII.
func (s *MCPService) RecordAudit(ctx context.Context, tenantID, actor, event string, payload map[string]any) error {
	values := make(map[string]any, len(payload)+3)
	for key, value := range payload {
		values[key] = value
	}
	values["tenant_id"] = tenantID
	values["actor"] = actor
	values["at"] = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode MCP audit: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_outbox (event_id, event_type, payload)
		VALUES (gen_random_uuid(), $1, $2::jsonb)
	`, event, encoded)
	if err != nil {
		return fmt.Errorf("insert MCP audit: %w", err)
	}
	return nil
}

func (s *MCPService) audit(ctx context.Context, tenantID, actor, event string, payload map[string]any) {
	_ = s.RecordAudit(ctx, tenantID, actor, event, payload)
}
