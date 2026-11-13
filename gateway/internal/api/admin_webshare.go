package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/webshare"
)

// AdminWebshareService exposes the Webshare integration.
type AdminWebshareService interface {
	Status(ctx context.Context, tenantID string) (admin.WebshareSettings, error)
	Test(ctx context.Context, tenantID, actor, apiKey, planID string) (admin.WebshareTestResult, error)
	Save(ctx context.Context, tenantID, actor string, input admin.WebshareSaveInput) (admin.WebshareSettings, error)
	Sync(ctx context.Context, tenantID, actor string) (admin.WebshareSyncResult, error)
	RefreshPool(ctx context.Context, tenantID string) error
}

func adminWebshareStatus(service AdminWebshareService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		status, err := service.Status(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load webshare status")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(status)
	}
}

func adminWebshareTest(service AdminWebshareService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			APIKey string `json:"api_key"`
			PlanID string `json:"plan_id"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		result, err := service.Test(request.Context(), principal.TenantID, principal.KeyID, input.APIKey, input.PlanID)
		if err != nil {
			writeWebshareError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func adminWebshareSave(service AdminWebshareService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.WebshareSaveInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		status, err := service.Save(request.Context(), principal.TenantID, principal.KeyID, input)
		if err != nil {
			writeWebshareError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(status)
	}
}

func adminWebshareSync(service AdminWebshareService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		result, err := service.Sync(request.Context(), principal.TenantID, principal.KeyID)
		if err != nil {
			if errors.Is(err, admin.ErrSyncInProgress) {
				writeAPIError(w, http.StatusConflict, "sync_in_progress", "webshare sync is already running")
				return
			}
			writeWebshareError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func writeWebshareError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, admin.ErrInvalidInput):
		writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "webshare api key is required")
	case errors.Is(err, webshare.ErrUnauthorized):
		writeAPIError(w, http.StatusBadRequest, "webshare_unauthorized", "invalid webshare api key")
	case errors.Is(err, webshare.ErrForbidden):
		writeAPIError(w, http.StatusBadRequest, "webshare_forbidden", "webshare plan access forbidden")
	case errors.Is(err, webshare.ErrRateLimited):
		writeAPIError(w, http.StatusTooManyRequests, "webshare_rate_limited", "webshare rate limit; retry in about 60 seconds")
	case errors.Is(err, webshare.ErrUnavailable):
		writeAPIError(w, http.StatusBadGateway, "webshare_unavailable", "webshare is unavailable")
	case errors.Is(err, webshare.ErrInvalidAnswer):
		writeAPIError(w, http.StatusBadGateway, "webshare_invalid_response", "invalid webshare response")
	default:
		writeAPIError(w, http.StatusBadGateway, "webshare_error", "webshare request failed")
	}
}

func adminWebshareRefresh(service AdminWebshareService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		if err := service.RefreshPool(request.Context(), principal.TenantID); err != nil {
			writeWebshareError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}
