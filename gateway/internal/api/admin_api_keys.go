package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminAPIKeyService exposes API key administration operations.
type AdminAPIKeyService interface {
	List(ctx context.Context, tenantID string) ([]admin.APIKey, error)
	Create(ctx context.Context, tenantID string, input admin.CreateAPIKeyInput) (admin.CreatedAPIKey, error)
	Revoke(ctx context.Context, tenantID, keyID string) (admin.APIKey, error)
}

func adminAPIKeysList(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		keys, err := service.List(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list API keys")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": keys})
	}
}

func adminAPIKeysCreate(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.CreateAPIKeyInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		created, err := service.Create(request.Context(), principal.TenantID, input)
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid API key input")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to create API key")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	}
}

func adminAPIKeysRevoke(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		key, err := service.Revoke(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "API key not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to revoke API key")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(key)
	}
}
