package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminUsageService exposes cost and observability aggregates.
type AdminUsageService interface {
	CostReport(ctx context.Context, tenantID string, days int) (admin.CostReport, error)
	Observability(ctx context.Context, tenantID string, days int) (admin.ObservabilitySummary, error)
}

func adminCostReport(service AdminUsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		report, err := service.CostReport(request.Context(), principal.TenantID, daysParam(request, 30))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load cost report")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(report)
	}
}

func adminObservabilitySummary(service AdminUsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		summary, err := service.Observability(request.Context(), principal.TenantID, daysParam(request, 1))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load observability summary")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

func daysParam(request *http.Request, fallback int) int {
	value, err := strconv.Atoi(request.URL.Query().Get("days"))
	if err != nil || value <= 0 || value > 365 {
		return fallback
	}
	return value
}
