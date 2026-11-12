package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminCatalogService exposes the versioned provider catalog.
type AdminCatalogService interface {
	ListCatalog(ctx context.Context) ([]admin.CatalogEntry, error)
	CreateFromCatalog(ctx context.Context, tenantID string, input admin.CreateFromCatalogInput) (admin.Provider, error)
}

func adminCatalogList(service AdminCatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if _, ok := requireManage(w, request); !ok {
			return
		}
		entries, err := service.ListCatalog(request.Context())
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list provider catalog")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": entries})
	}
}

func adminCatalogConnect(service AdminCatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.CreateFromCatalogInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		input.Slug = request.PathValue("slug")
		provider, err := service.CreateFromCatalog(request.Context(), principal.TenantID, input)
		switch {
		case errors.Is(err, admin.ErrNotFound):
			writeAPIError(w, http.StatusNotFound, "not_found", "catalog provider not found")
			return
		case errors.Is(err, admin.ErrInvalidInput):
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid catalog provider input")
			return
		case errors.Is(err, admin.ErrConflict):
			writeAPIError(w, http.StatusConflict, "conflict_error", "provider already exists")
			return
		case err != nil:
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to connect catalog provider")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(provider)
	}
}
