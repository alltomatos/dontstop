package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

func adminProviderConfiguredModels(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		models, err := service.ConfiguredModels(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list provider models")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": models})
	}
}

func adminProviderSyncModels(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		result, err := service.SyncModels(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusBadGateway, "api_error", "failed to sync provider models")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func adminProviderSetModelActive(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			ModelID string `json:"model_id"`
			Active  *bool  `json:"active"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		if input.Active == nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "active is required")
			return
		}
		err := service.SetModelActive(request.Context(), principal.TenantID, request.PathValue("id"), input.ModelID, *input.Active)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider model not found")
			return
		}
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid model input")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update provider model")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"model_id": input.ModelID, "active": *input.Active})
	}
}

func adminProviderTestModel(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			ModelID   string `json:"model_id"`
			AccountID string `json:"account_id"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		result, err := service.TestModel(request.Context(), principal.TenantID, request.PathValue("id"), input.ModelID, input.AccountID)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider model not found")
			return
		}
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid model input")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to test provider model")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func adminProviderStartModelTestRun(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			Models []string `json:"models"`
		}
		if request.ContentLength > 0 {
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
				return
			}
		}
		run, err := service.StartModelTestRun(request.Context(), principal.TenantID, request.PathValue("id"), input.Models)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "provider has no models to test")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to start model test run")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(run)
	}
}

func adminProviderLatestModelTestRun(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		run, err := service.LatestModelTestRun(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "no model test run found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load model test run")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(run)
	}
}

func adminProviderModelTestRun(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		run, err := service.GetModelTestRun(request.Context(), principal.TenantID, request.PathValue("id"), request.PathValue("runId"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "model test run not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load model test run")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(run)
	}
}

func adminProviderBulkSetModelActive(service AdminProviderService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			Models []string `json:"models"`
			Active *bool    `json:"active"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		if input.Active == nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "active is required")
			return
		}
		results, err := service.SetModelsActive(request.Context(), principal.TenantID, request.PathValue("id"), input.Models, *input.Active)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "models are required")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update provider models")
			return
		}
		updated := 0
		for _, result := range results {
			if result.OK {
				updated++
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"updated": updated, "results": results})
	}
}
