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
	Readiness *health.Registry
	Verifier  *auth.Verifier
	Models    ModelLister
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
		XDonstopStatus string `json:"x-dontstop-status"`
	}
	data := make([]modelObject, 0, len(models))
	for _, model := range models {
		data = append(data, modelObject{
			ID:             model.ID,
			Object:         "model",
			Created:        model.CreatedAt,
			OwnedBy:        model.OwnedBy,
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
