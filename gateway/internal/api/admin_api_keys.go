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
	Get(ctx context.Context, tenantID, keyID string) (admin.APIKey, error)
	Create(ctx context.Context, tenantID, actor string, input admin.CreateAPIKeyInput) (admin.CreatedAPIKey, error)
	Update(ctx context.Context, tenantID, actor, keyID string, input admin.UpdateAPIKeyInput) (admin.APIKey, error)
	Rotate(ctx context.Context, tenantID, actor, keyID string) (admin.CreatedAPIKey, error)
	Revoke(ctx context.Context, tenantID, actor, keyID string) (admin.APIKey, error)
	Scopes() []string
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
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": keys})
	}
}

func adminAPIKeyScopes(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if _, ok := requireManage(w, request); !ok {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": service.Scopes()})
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
		created, err := service.Create(request.Context(), principal.TenantID, principal.KeyID, input)
		if writeAPIKeyError(w, err, "failed to create API key") {
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func adminAPIKeysGet(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		key, err := service.Get(request.Context(), principal.TenantID, request.PathValue("id"))
		if writeAPIKeyError(w, err, "failed to load API key") {
			return
		}
		writeJSON(w, http.StatusOK, key)
	}
}

func adminAPIKeysUpdate(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.UpdateAPIKeyInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		key, err := service.Update(request.Context(), principal.TenantID, principal.KeyID, request.PathValue("id"), input)
		if writeAPIKeyError(w, err, "failed to update API key") {
			return
		}
		writeJSON(w, http.StatusOK, key)
	}
}

func adminAPIKeysRotate(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		rotated, err := service.Rotate(request.Context(), principal.TenantID, principal.KeyID, request.PathValue("id"))
		if writeAPIKeyError(w, err, "failed to rotate API key") {
			return
		}
		writeJSON(w, http.StatusOK, rotated)
	}
}

func adminAPIKeysRevoke(service AdminAPIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		key, err := service.Revoke(request.Context(), principal.TenantID, principal.KeyID, request.PathValue("id"))
		if writeAPIKeyError(w, err, "failed to revoke API key") {
			return
		}
		writeJSON(w, http.StatusOK, key)
	}
}

func writeAPIKeyError(w http.ResponseWriter, err error, fallback string) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, admin.ErrInvalidInput):
		writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid API key input")
	case errors.Is(err, admin.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "not_found", "API key not found")
	case errors.Is(err, admin.ErrConflict):
		writeAPIError(w, http.StatusConflict, "conflict_error", "API key cannot be changed in its current state")
	default:
		writeAPIError(w, http.StatusInternalServerError, "api_error", fallback)
	}
	return true
}
