package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

type fakeAdminUsage struct{}

func (fakeAdminUsage) CostReport(context.Context, string, int) (admin.CostReport, error) {
	return admin.CostReport{Summary: admin.CostSummary{Days: 30, CostUSD: 12.5, Requests: 4}}, nil
}

func (fakeAdminUsage) Observability(context.Context, string, int) (admin.ObservabilitySummary, error) {
	return admin.ObservabilitySummary{Requests: 10, Errors: 1, ErrorRate: 0.1, P95MS: 28}, nil
}

func TestAdminUsageEndpoints(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness:  health.NewRegistry(0),
		Verifier:   verifier,
		AdminUsage: fakeAdminUsage{},
	})
	for path, expected := range map[string]string{
		"/admin/v1/costs/summary?days=30":        `"cost_usd":12.5`,
		"/admin/v1/observability/summary?days=1": `"p95_ms":28`,
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("%s: unexpected response %d %s", path, response.Code, response.Body.String())
		}
	}
}
