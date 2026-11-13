package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminComboService exposes combo administration.
type AdminComboService interface {
	List(ctx context.Context, tenantID string) ([]admin.Combo, error)
	Get(ctx context.Context, tenantID, comboID string) (admin.Combo, error)
	Create(ctx context.Context, tenantID string, input admin.ComboInput) (admin.Combo, error)
	Update(ctx context.Context, tenantID, comboID string, revision int, input admin.ComboInput) (admin.Combo, error)
	SetStatus(ctx context.Context, tenantID, comboID, status string) (admin.Combo, error)
	Delete(ctx context.Context, tenantID, comboID string) error
	Catalog(ctx context.Context, tenantID string) ([]admin.ComboCatalogModel, error)
	ResolveTargets(ctx context.Context, tenantID, slug string) ([]admin.ComboTarget, int, error)
}

// AdminComboAIService exposes the catalog-assisted draft endpoint.
type AdminComboAIService interface {
	Draft(ctx context.Context, tenantID string, input admin.ComboAIInput) (admin.ComboAIDraft, error)
}

func adminCombosList(service AdminComboService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		combos, err := service.List(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to list combos")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": combos})
	}
}

func adminComboGet(service AdminComboService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		combo, err := service.Get(request.Context(), principal.TenantID, request.PathValue("id"))
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "combo not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load combo")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(combo)
	}
}

func adminComboCreate(service AdminComboService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.ComboInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		combo, err := service.Create(request.Context(), principal.TenantID, input)
		if writeComboError(w, err) {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(combo)
	}
}

func adminComboUpdate(service AdminComboService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		revision, err := strconv.Atoi(request.URL.Query().Get("revision"))
		if err != nil || revision < 1 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "revision is required")
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.ComboInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		combo, err := service.Update(request.Context(), principal.TenantID, request.PathValue("id"), revision, input)
		if writeComboError(w, err) {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(combo)
	}
}

func adminComboStatus(service AdminComboService, status string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		combo, err := service.SetStatus(request.Context(), principal.TenantID, request.PathValue("id"), status)
		if writeComboError(w, err) {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(combo)
	}
}

func adminComboDelete(service AdminComboService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		if err := service.Delete(request.Context(), principal.TenantID, request.PathValue("id")); err != nil {
			if writeComboError(w, err) {
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
	}
}

func adminComboCatalog(service AdminComboService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		models, err := service.Catalog(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load combo catalog")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": models})
	}
}

func adminComboAIDraft(service AdminComboAIService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.ComboAIInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		draft, err := service.Draft(request.Context(), principal.TenantID, input)
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid AI request")
			return
		}
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "assistant model not available")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusBadGateway, "ai_error", "AI draft failed")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(draft)
	}
}

func writeComboError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, admin.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "not_found", "combo not found")
	case errors.Is(err, admin.ErrInvalidInput):
		writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid combo input")
	case errors.Is(err, admin.ErrConflict):
		writeAPIError(w, http.StatusConflict, "conflict_error", "combo conflict or no usable member")
	default:
		writeAPIError(w, http.StatusInternalServerError, "api_error", "combo operation failed")
	}
	return true
}
