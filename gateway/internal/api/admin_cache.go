package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminCacheService exposes cache aggregates.
type AdminCacheService interface {
	Stats(ctx context.Context, tenantID string, days int) (admin.CacheStats, error)
	Events(ctx context.Context, tenantID string, limit int) ([]admin.CacheEvent, error)
}

func adminCacheStats(service AdminCacheService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		stats, err := service.Stats(request.Context(), principal.TenantID, daysParam(request, 1))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load cache stats")
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}

func adminCacheEvents(service AdminCacheService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = 50
		}
		events, err := service.Events(request.Context(), principal.TenantID, limit)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load cache events")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": events})
	}
}
