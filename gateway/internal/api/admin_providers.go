package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
)

// AdminProviderService exposes provider administration operations.
type AdminProviderService interface {
	List(ctx context.Context, tenantID string) ([]admin.Provider, error)
	Get(ctx context.Context, tenantID, providerID string) (admin.Provider, error)
	Create(ctx context.Context, tenantID string, input admin.CreateProviderInput) (admin.Provider, error)
	AddAccount(ctx context.Context, tenantID, providerID string, input admin.AccountInput) (admin.Account, error)
}

func adminProvidersList(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		providers, err := service.List(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list providers")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": providers})
	}
}

func adminProviderGet(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		provider, err := service.Get(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load provider")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(provider)
	}
}

func adminProviderCreate(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.CreateProviderInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		provider, err := service.Create(request.Context(), principal.TenantID, input)
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid provider input")
			return
		}
		if errors.Is(err, admin.ErrConflict) {
			writeAPIError(w, http.StatusConflict, "conflict_error", "provider already exists")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to create provider")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(provider)
	}
}

func adminProviderAddAccount(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.AccountInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		account, err := service.AddAccount(request.Context(), principal.TenantID, request.PathValue("id"), input)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid account input")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to add account")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(account)
	}
}

func requireManage(w http.ResponseWriter, request *http.Request) (auth.Principal, bool) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok || !principal.HasScope("manage") {
		writeAPIError(w, http.StatusForbidden, "permission_error", "missing manage scope")
		return auth.Principal{}, false
	}
	return principal, true
}
