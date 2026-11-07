package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

// ModelLister returns the models available to a tenant.
type ModelLister interface {
	ListModels(ctx context.Context, tenantID string) ([]provider.Model, error)
}

// Dependencies contains the gateway HTTP collaborators.
type Dependencies struct {
	Readiness  *health.Registry
	Verifier   *auth.Verifier
	Models     ModelLister
	Chat       ChatService
	Admin      AdminProviderService
	AdminKeys  AdminAPIKeyService
	AdminUsage AdminUsageService
	AdminMCP   AdminMCPService
}

// NewRouter builds the public HTTP surface for the gateway.
func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	if deps.Readiness != nil {
		mux.Handle("GET /healthz", deps.Readiness.Liveness())
		mux.Handle("GET /readyz", deps.Readiness.Readiness())
	}
	if deps.Verifier != nil && deps.Models != nil {
		mux.Handle("GET /v1/models", deps.Verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			listModels(w, request, deps.Models)
		})))
	}
	if deps.Verifier != nil && deps.Chat != nil {
		mux.Handle("POST /v1/chat/completions", deps.Verifier.Middleware(chatCompletions(deps.Chat)))
	}
	if deps.Verifier != nil && deps.Admin != nil {
		mux.Handle("GET /admin/v1/providers", deps.Verifier.Middleware(adminProvidersList(deps.Admin)))
		mux.Handle("GET /admin/v1/providers/{id}", deps.Verifier.Middleware(adminProviderGet(deps.Admin)))
		mux.Handle("POST /admin/v1/providers/{id}/accounts", deps.Verifier.Middleware(adminProviderAddAccount(deps.Admin)))
	}
	if deps.Verifier != nil && deps.AdminKeys != nil {
		mux.Handle("GET /admin/v1/api-keys", deps.Verifier.Middleware(adminAPIKeysList(deps.AdminKeys)))
		mux.Handle("POST /admin/v1/api-keys", deps.Verifier.Middleware(adminAPIKeysCreate(deps.AdminKeys)))
		mux.Handle("POST /admin/v1/api-keys/{id}/revoke", deps.Verifier.Middleware(adminAPIKeysRevoke(deps.AdminKeys)))
	}
	if deps.Verifier != nil && deps.AdminUsage != nil {
		mux.Handle("GET /admin/v1/costs/summary", deps.Verifier.Middleware(adminCostReport(deps.AdminUsage)))
		mux.Handle("GET /admin/v1/observability/summary", deps.Verifier.Middleware(adminObservabilitySummary(deps.AdminUsage)))
	}
	if deps.Verifier != nil && deps.AdminMCP != nil {
		mux.Handle("GET /admin/v1/mcp/status", deps.Verifier.Middleware(adminMCPStatus(deps.AdminMCP)))
		mux.Handle("PATCH /admin/v1/mcp/settings", deps.Verifier.Middleware(adminMCPUpdateSettings(deps.AdminMCP)))
		mux.Handle("PATCH /admin/v1/mcp/tools/{id}", deps.Verifier.Middleware(adminMCPUpdateTool(deps.AdminMCP)))
	}
	return mux
}

func listModels(w http.ResponseWriter, request *http.Request, lister ModelLister) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok || !principal.HasScope("models:read") {
		writeAPIError(w, http.StatusForbidden, "permission_error", "missing models:read scope")
		return
	}
	models, err := lister.ListModels(request.Context(), principal.TenantID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list models")
		return
	}

	type modelObject struct {
		ID             string `json:"id"`
		Object         string `json:"object"`
		Created        int64  `json:"created"`
		OwnedBy        string `json:"owned_by"`
		Origin         string `json:"x-dontstop-origin"`
		XDonstopStatus string `json:"x-dontstop-status"`
	}
	data := make([]modelObject, 0, len(models))
	for _, model := range models {
		data = append(data, modelObject{
			ID:             model.ID,
			Object:         "model",
			Created:        model.CreatedAt,
			OwnedBy:        model.OwnedBy,
			Origin:         model.Origin,
			XDonstopStatus: model.Status,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   data,
	})
}

func writeAPIError(w http.ResponseWriter, status int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"type":    errorType,
			"message": message,
		},
	})
}
