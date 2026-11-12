package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminProxyService exposes the proxy pool and account assignments.
type AdminProxyService interface {
	List(ctx context.Context, tenantID string) ([]admin.ProxyEndpoint, error)
	Create(ctx context.Context, tenantID string, input admin.CreateProxyEndpointInput) (admin.ProxyEndpoint, error)
	Assign(ctx context.Context, tenantID, providerID, accountID, endpointID string) error
	Unassign(ctx context.Context, tenantID, providerID, accountID string) error
}

func adminProxiesList(service AdminProxyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		endpoints, err := service.List(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list proxies")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": endpoints})
	}
}

func adminProxyCreate(service AdminProxyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.CreateProxyEndpointInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		endpoint, err := service.Create(request.Context(), principal.TenantID, input)
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid proxy input")
			return
		}
		if errors.Is(err, admin.ErrConflict) {
			writeAPIError(w, http.StatusConflict, "conflict_error", "proxy already exists")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to create proxy")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(endpoint)
	}
}

func adminProviderSetProxyMode(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			Name      *string `json:"name"`
			ProxyMode *string `json:"proxy_mode"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		if input.Name == nil && input.ProxyMode == nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "name or proxy_mode is required")
			return
		}
		providerID := request.PathValue("id")
		accountID := request.PathValue("accountId")
		if input.Name != nil {
			err := service.RenameAccount(request.Context(), principal.TenantID, providerID, accountID, *input.Name)
			if errors.Is(err, admin.ErrNotFound) {
				writeAPIError(w, http.StatusNotFound, "not_found", "provider account not found")
				return
			}
			if errors.Is(err, admin.ErrConflict) {
				writeAPIError(w, http.StatusConflict, "conflict_error", "account name already exists")
				return
			}
			if errors.Is(err, admin.ErrInvalidInput) {
				writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid account name")
				return
			}
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to rename account")
				return
			}
		}
		if input.ProxyMode != nil {
			mode, err := service.SetProxyMode(request.Context(), principal.TenantID, providerID, accountID, *input.ProxyMode)
			if errors.Is(err, admin.ErrNotFound) {
				writeAPIError(w, http.StatusNotFound, "not_found", "provider account not found")
				return
			}
			if errors.Is(err, admin.ErrInvalidInput) {
				writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "proxy_mode must be off, preferred or required")
				return
			}
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update proxy mode")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]string{"proxy_mode": mode})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func adminProviderAccountSummary(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		result, err := service.AccountSummary(
			request.Context(), principal.TenantID,
			request.PathValue("id"), request.PathValue("accountId"),
		)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider account not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load provider account summary")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func adminProviderTestAccount(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		result, err := service.TestAccount(
			request.Context(), principal.TenantID,
			request.PathValue("id"), request.PathValue("accountId"),
		)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider account not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to test provider account")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func adminProviderAssignProxy(service AdminProxyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			ProxyEndpointID string `json:"proxy_endpoint_id"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		err := service.Assign(
			request.Context(), principal.TenantID,
			request.PathValue("id"), request.PathValue("accountId"), input.ProxyEndpointID,
		)
		writeProxyAssignmentError(w, err)
	}
}

func adminProviderUnassignProxy(service AdminProxyService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		err := service.Unassign(
			request.Context(), principal.TenantID,
			request.PathValue("id"), request.PathValue("accountId"),
		)
		writeProxyAssignmentError(w, err)
	}
}

func writeProxyAssignmentError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	case errors.Is(err, admin.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "not_found", "provider account or proxy not found")
	case errors.Is(err, admin.ErrProxyLimit):
		writeAPIError(w, http.StatusConflict, "proxy_limit_error", "proxy already serves the maximum of 3 accounts")
	case errors.Is(err, admin.ErrInvalidInput):
		writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid proxy assignment")
	default:
		writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update proxy assignment")
	}
}
