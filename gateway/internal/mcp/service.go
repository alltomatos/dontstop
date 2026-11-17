package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/mcp/catalog"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ChatService is the subset of the chat service used by MCP tools.
type ChatService interface {
	Chat(ctx context.Context, tenantID string, request provider.ChatRequest, idempotencyKey string) (provider.ChatResponse, string, error)
}

// ModelLister lists models available to a tenant.
type ModelLister interface {
	ListModels(ctx context.Context, tenantID string) ([]provider.Model, error)
}

// UsageService returns tenant usage aggregates.
type UsageService interface {
	CostReport(ctx context.Context, tenantID string, days int) (admin.CostReport, error)
}

// CacheService returns tenant cache aggregates.
type CacheService interface {
	Stats(ctx context.Context, tenantID string, days int) (admin.CacheStats, error)
	ByDay(ctx context.Context, tenantID string, days int) ([]admin.CacheDay, error)
}

// ComboService is the subset of combo administration used by MCP tools.
type ComboService interface {
	List(ctx context.Context, tenantID string) ([]admin.Combo, error)
	Get(ctx context.Context, tenantID, comboID string) (admin.Combo, error)
	Create(ctx context.Context, tenantID string, input admin.ComboInput) (admin.Combo, error)
	Update(ctx context.Context, tenantID, comboID string, revision int, input admin.ComboInput) (admin.Combo, error)
	SetStatus(ctx context.Context, tenantID, comboID, status string) (admin.Combo, error)
	Delete(ctx context.Context, tenantID, comboID string) error
	Catalog(ctx context.Context, tenantID string) ([]admin.ComboCatalogModel, error)
}

// AdminStore persists settings and audit events for the runtime.
type AdminStore interface {
	Settings(ctx context.Context, tenantID string) (admin.MCPConfig, error)
	ToolOverrides(ctx context.Context, tenantID string) (map[string]bool, error)
	RecordAudit(ctx context.Context, tenantID, actor, event string, payload map[string]any) error
}

// Dependencies contains the MCP runtime collaborators.
type Dependencies struct {
	Store  AdminStore
	Chat   ChatService
	Models ModelLister
	Usage  UsageService
	Cache  CacheService
	Combos ComboService
	Logger *slog.Logger
}

// Service is the tenant-aware MCP runtime.
type Service struct {
	store  AdminStore
	chat   ChatService
	models ModelLister
	usage  UsageService
	cache  CacheService
	combos ComboService
	logger *slog.Logger
}

// NewService creates an MCP runtime.
func NewService(deps Dependencies) *Service {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:  deps.Store,
		chat:   deps.Chat,
		models: deps.Models,
		usage:  deps.Usage,
		cache:  deps.Cache,
		combos: deps.Combos,
		logger: logger,
	}
}

// AllowedTools returns enabled tools the principal may call.
func (s *Service) AllowedTools(ctx context.Context, principal auth.Principal) ([]catalog.Tool, error) {
	if s.store == nil {
		return nil, errors.New("MCP settings store is not configured")
	}
	overrides, err := s.store.ToolOverrides(ctx, principal.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load MCP tool settings: %w", err)
	}
	return filterTools(catalog.Tools, overrides, principal.Scopes), nil
}

func filterTools(tools []catalog.Tool, overrides map[string]bool, scopes []string) []catalog.Tool {
	allowed := make([]catalog.Tool, 0, len(tools))
	for _, tool := range tools {
		if enabled, ok := overrides[tool.ID]; ok && !enabled {
			continue
		}
		if tool.Scope != "" && !hasScope(scopes, tool.Scope) {
			continue
		}
		allowed = append(allowed, tool)
	}
	return allowed
}

func hasScope(scopes []string, scope string) bool {
	for _, granted := range scopes {
		if granted == scope {
			return true
		}
	}
	return false
}

// NewServer builds a server exposing only the provided tools for one principal.
func (s *Service) NewServer(principal auth.Principal, tools []catalog.Tool) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:        "dontstop",
		Title:       "dontstop MCP",
		Description: "Tenant-scoped dontstop gateway tools.",
		Version:     "0.1.0",
	}, &sdkmcp.ServerOptions{
		Instructions: "Tools are scoped by the API key and tenant. Secrets are never exposed.",
		Logger:       s.logger,
	})
	s.registerTools(server, principal, tools)
	return server
}

// StreamableHandler serves the MCP Streamable HTTP transport at /mcp.
func (s *Service) StreamableHandler() http.Handler {
	return sdkmcp.NewStreamableHTTPHandler(s.serverForRequest, &sdkmcp.StreamableHTTPOptions{
		Stateless: true,
		Logger:    s.logger,
	})
}

// SSEHandler serves the legacy SSE transport.
func (s *Service) SSEHandler() http.Handler {
	return sdkmcp.NewSSEHandler(s.serverForRequest, nil)
}

// Gate rejects requests when MCP is disabled or the configured transport differs.
func (s *Service) Gate(transport string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if s.store == nil {
			http.NotFound(w, request)
			return
		}
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		config, err := s.store.Settings(request.Context(), principal.TenantID)
		if err != nil || !config.Enabled || config.Transport != transport {
			http.NotFound(w, request)
			return
		}
		next.ServeHTTP(w, request)
	})
}

// RunStdio serves MCP over stdin/stdout for one pre-authenticated principal.
func (s *Service) RunStdio(ctx context.Context, principal auth.Principal) error {
	if s.store == nil {
		return errors.New("MCP settings store is not configured")
	}
	config, err := s.store.Settings(ctx, principal.TenantID)
	if err != nil {
		return fmt.Errorf("load MCP settings: %w", err)
	}
	if !config.Enabled || config.Transport != "stdio" {
		return errors.New("MCP stdio is disabled for this tenant")
	}
	tools, err := s.AllowedTools(ctx, principal)
	if err != nil {
		return err
	}
	return s.NewServer(principal, tools).Run(ctx, &sdkmcp.StdioTransport{})
}

func (s *Service) serverForRequest(request *http.Request) *sdkmcp.Server {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		return nil
	}
	tools, err := s.AllowedTools(request.Context(), principal)
	if err != nil {
		s.logger.Warn("mcp load tools failed", "tenant_id", principal.TenantID, "error", err)
		return nil
	}
	return s.NewServer(principal, tools)
}
