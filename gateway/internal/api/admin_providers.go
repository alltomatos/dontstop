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
	DeleteAccount(ctx context.Context, tenantID, providerID, accountID string) (bool, error)
	SetProxyMode(ctx context.Context, tenantID, providerID, accountID, raw string) (string, error)
	RenameAccount(ctx context.Context, tenantID, providerID, accountID, name string) error
	TestAccount(ctx context.Context, tenantID, providerID, accountID string) (admin.AccountTestResult, error)
	AccountSummary(ctx context.Context, tenantID, providerID, accountID string) (admin.AccountSummaryResult, error)
	Test(ctx context.Context, tenantID, providerID string) (admin.ProviderTestResult, error)
	RemoteModels(ctx context.Context, tenantID, providerID string) ([]admin.RemoteModel, error)
	ImportModels(ctx context.Context, tenantID, providerID string, ids []string) (int, error)
	ConfiguredModels(ctx context.Context, tenantID, providerID string) ([]admin.ConfiguredModel, error)
	SyncModels(ctx context.Context, tenantID, providerID string) (admin.SyncModelsResult, error)
	SetModelActive(ctx context.Context, tenantID, providerID, modelID string, active bool) error
	SetModelsActive(ctx context.Context, tenantID, providerID string, models []string, active bool) ([]admin.ModelUpdateResult, error)
	TestModel(ctx context.Context, tenantID, providerID, modelID, accountID string) (admin.ModelTestResult, error)
	StartModelTestRun(ctx context.Context, tenantID, providerID string, models []string) (admin.ModelTestRun, error)
	GetModelTestRun(ctx context.Context, tenantID, providerID, runID string) (admin.ModelTestRun, error)
	LatestModelTestRun(ctx context.Context, tenantID, providerID string) (admin.ModelTestRun, error)
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

func adminProviderDeleteAccount(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		providerDeleted, err := service.DeleteAccount(request.Context(), principal.TenantID, request.PathValue("id"), request.PathValue("accountId"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider account not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to delete provider account")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{
			"deleted":          true,
			"provider_deleted": providerDeleted,
		})
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

func adminProviderTest(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		result, err := service.Test(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to test provider")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func adminProviderModels(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		models, err := service.RemoteModels(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if errors.Is(err, admin.ErrModelsUnsupported) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "provider does not expose a models endpoint")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusBadGateway, "api_error", "failed to list provider models")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": models})
	}
}

func adminProviderImportModels(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			Models []string `json:"models"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		if len(input.Models) == 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "models are required")
			return
		}
		imported, err := service.ImportModels(request.Context(), principal.TenantID, request.PathValue("id"), input.Models)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if errors.Is(err, admin.ErrModelsUnsupported) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "provider does not expose a models endpoint")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusBadGateway, "api_error", "failed to import provider models")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"imported": imported})
	}
}
