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

type fakeAdminCache struct{}

func (fakeAdminCache) Stats(context.Context, string, int) (admin.CacheStats, error) {
	return admin.CacheStats{Hits: 8, Misses: 2, Requests: 10, HitRate: 0.8}, nil
}

func (fakeAdminCache) Events(context.Context, string, int) ([]admin.CacheEvent, error) {
	return []admin.CacheEvent{{Time: "10:44", Status: "hit", Model: "gpt-4.1", LatencyMS: 12, Tokens: 1200}}, nil
}

func TestAdminCacheEndpoints(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness:  health.NewRegistry(0),
		Verifier:   verifier,
		AdminCache: fakeAdminCache{},
	})
	for path, expected := range map[string]string{
		"/admin/v1/cache/stats?days=1":    `"hit_rate":0.8`,
		"/admin/v1/cache/events?limit=10": `"status":"hit"`,
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
