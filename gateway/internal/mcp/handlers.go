package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/mcp/catalog"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type emptyInput struct{}

type chatInput struct {
	Model       string                 `json:"model" jsonschema:"model ID or combo/<slug>"`
	Messages    []provider.ChatMessage `json:"messages" jsonschema:"chat messages"`
	Temperature *float64               `json:"temperature,omitempty" jsonschema:"sampling temperature"`
	MaxTokens   int                    `json:"max_tokens,omitempty" jsonschema:"maximum completion tokens"`
}

type chatOutput struct {
	Model        string         `json:"model"`
	Content      string         `json:"content"`
	FinishReason string         `json:"finish_reason,omitempty"`
	Usage        provider.Usage `json:"usage"`
	CacheStatus  string         `json:"cache_status,omitempty"`
}

type modelView struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
	OwnedBy     string `json:"owned_by,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Status      string `json:"status,omitempty"`
	CreatedAt   int64  `json:"created_at,omitempty"`
}

type usageInput struct {
	Days int `json:"days,omitempty" jsonschema:"number of days (default 30)"`
}

type toolIDInput struct {
	ID string `json:"id" jsonschema:"resource ID"`
}

func (s *Service) registerTools(server *sdkmcp.Server, principal auth.Principal, tools []catalog.Tool) {
	for _, tool := range tools {
		switch tool.ID {
		case "dontstop_chat":
			auditedTool(server, s, principal, tool, s.chatHandler(principal.TenantID))
		case "dontstop_models":
			auditedTool(server, s, principal, tool, s.modelsHandler(principal.TenantID))
		case "dontstop_embeddings":
			auditedTool(server, s, principal, tool, unavailableHandler[emptyInput]("embeddings backend is not configured"))
		case "dontstop_web_search":
			auditedTool(server, s, principal, tool, unavailableHandler[emptyInput]("web search backend is not configured"))
		case "dontstop_usage":
			auditedTool(server, s, principal, tool, s.usageHandler(principal.TenantID))
		case "dontstop_cache":
			auditedTool(server, s, principal, tool, s.cacheHandler(principal.TenantID))
		case "dontstop_health":
			auditedTool(server, s, principal, tool, healthHandler)
		case "dontstop_combo_catalog":
			auditedTool(server, s, principal, tool, s.comboCatalogHandler(principal.TenantID))
		case "dontstop_combo_list":
			auditedTool(server, s, principal, tool, s.comboListHandler(principal.TenantID))
		case "dontstop_combo_get":
			auditedTool(server, s, principal, tool, s.comboGetHandler(principal.TenantID))
		case "dontstop_combo_create":
			auditedTool(server, s, principal, tool, s.comboCreateHandler(principal.TenantID, principal.KeyID))
		case "dontstop_combo_update":
			auditedTool(server, s, principal, tool, s.comboUpdateHandler(principal.TenantID, principal.KeyID))
		case "dontstop_combo_activate":
			auditedTool(server, s, principal, tool, s.comboStatusHandler(principal.TenantID, principal.KeyID, "active"))
		case "dontstop_combo_deactivate":
			auditedTool(server, s, principal, tool, s.comboStatusHandler(principal.TenantID, principal.KeyID, "inactive"))
		case "dontstop_combo_delete":
			auditedTool(server, s, principal, tool, s.comboDeleteHandler(principal.TenantID, principal.KeyID))
		case "dontstop_provider_list":
			auditedTool(server, s, principal, tool, s.providerListHandler(principal.TenantID))
		}
	}
}

func auditedTool[In, Out any](server *sdkmcp.Server, service *Service, principal auth.Principal, tool catalog.Tool, handler sdkmcp.ToolHandlerFor[In, Out]) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		Annotations: annotations(tool),
	}, auditHandler(service, principal, tool, handler))
}

func annotations(tool catalog.Tool) *sdkmcp.ToolAnnotations {
	destructive := tool.Destructive
	openWorld := tool.OpenWorld
	return &sdkmcp.ToolAnnotations{
		ReadOnlyHint:    tool.ReadOnly,
		DestructiveHint: &destructive,
		IdempotentHint:  tool.Idempotent,
		OpenWorldHint:   &openWorld,
	}
}

func auditHandler[In, Out any](service *Service, principal auth.Principal, tool catalog.Tool, handler sdkmcp.ToolHandlerFor[In, Out]) sdkmcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, request *sdkmcp.CallToolRequest, input In) (*sdkmcp.CallToolResult, Out, error) {
		started := time.Now()
		result, output, err := handler(ctx, request, input)
		status := "ok"
		if err != nil || (result != nil && result.IsError) {
			status = "error"
		}
		if service.store != nil {
			auditErr := service.store.RecordAudit(ctx, principal.TenantID, principal.KeyID, "mcp.tool.call", map[string]any{
				"tool":        tool.ID,
				"status":      status,
				"duration_ms": time.Since(started).Milliseconds(),
			})
			if auditErr != nil {
				service.logger.Warn("mcp audit failed", "tool", tool.ID, "error", auditErr)
			}
		}
		return result, output, err
	}
}

func (s *Service) chatHandler(tenantID string) sdkmcp.ToolHandlerFor[chatInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input chatInput) (*sdkmcp.CallToolResult, any, error) {
		if s.chat == nil {
			return nil, nil, errors.New("chat service is not configured")
		}
		response, cacheStatus, err := s.chat.Chat(ctx, tenantID, provider.ChatRequest{
			Model:       input.Model,
			Messages:    input.Messages,
			Temperature: input.Temperature,
			MaxTokens:   input.MaxTokens,
		}, "")
		if err != nil {
			s.logger.Warn("mcp chat failed", "tenant_id", tenantID)
			return nil, nil, errors.New("chat completion failed")
		}
		return toolResult(chatOutput{
			Model:        response.Model,
			Content:      response.Content,
			FinishReason: response.FinishReason,
			Usage:        response.Usage,
			CacheStatus:  cacheStatus,
		})
	}
}

func (s *Service) modelsHandler(tenantID string) sdkmcp.ToolHandlerFor[emptyInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, any, error) {
		if s.models == nil {
			return nil, nil, errors.New("model catalog is not configured")
		}
		models, err := s.models.ListModels(ctx, tenantID)
		if err != nil {
			s.logger.Warn("mcp models failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to list models")
		}
		views := make([]modelView, 0, len(models))
		for _, model := range models {
			views = append(views, modelView{
				ID: model.ID, DisplayName: model.DisplayName, OwnedBy: model.OwnedBy,
				Origin: model.Origin, Status: model.Status, CreatedAt: model.CreatedAt,
			})
		}
		return toolResult(views)
	}
}

func (s *Service) usageHandler(tenantID string) sdkmcp.ToolHandlerFor[usageInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input usageInput) (*sdkmcp.CallToolResult, any, error) {
		if s.usage == nil {
			return nil, nil, errors.New("usage service is not configured")
		}
		days := input.Days
		if days <= 0 {
			days = 30
		}
		report, err := s.usage.CostReport(ctx, tenantID, days)
		if err != nil {
			s.logger.Warn("mcp usage failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to load usage")
		}
		return toolResult(report)
	}
}

// cacheReport is the MCP cache payload: aggregates plus the daily series.
type cacheReport struct {
	Stats admin.CacheStats `json:"stats"`
	ByDay []admin.CacheDay `json:"by_day"`
}

func (s *Service) cacheHandler(tenantID string) sdkmcp.ToolHandlerFor[usageInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input usageInput) (*sdkmcp.CallToolResult, any, error) {
		if s.cache == nil {
			return nil, nil, errors.New("cache service is not configured")
		}
		days := input.Days
		if days <= 0 {
			days = 7
		}
		stats, err := s.cache.Stats(ctx, tenantID, days)
		if err != nil {
			s.logger.Warn("mcp cache stats failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to load cache stats")
		}
		series, err := s.cache.ByDay(ctx, tenantID, days)
		if err != nil {
			s.logger.Warn("mcp cache series failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to load cache series")
		}
		return toolResult(cacheReport{Stats: stats, ByDay: series})
	}
}

func healthHandler(_ context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, any, error) {	return toolResult(map[string]any{"status": "ok"})
}

func unavailableHandler[In any](message string) sdkmcp.ToolHandlerFor[In, any] {
	return func(context.Context, *sdkmcp.CallToolRequest, In) (*sdkmcp.CallToolResult, any, error) {
		return nil, nil, errors.New(message)
	}
}

func toolResult(value any) (*sdkmcp.CallToolResult, any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, nil, errors.New("failed to encode tool result")
	}
	return &sdkmcp.CallToolResult{
		Content:           []sdkmcp.Content{&sdkmcp.TextContent{Text: string(encoded)}},
		StructuredContent: value,
	}, value, nil
}
