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
	return admin.CacheStats{
		Days: 1, Hits: 8, Misses: 2, Bypass: 5, Cacheable: 10, Requests: 15,
		HitRate: 0.8, HitLatencyMS: 4, MissLatencyMS: 120, LatencySavedMS: 116, TokensReused: 1200,
	}, nil
}

func (fakeAdminCache) ByDay(context.Context, string, int) ([]admin.CacheDay, error) {
	return []admin.CacheDay{{Day: "2026-11-16", Hits: 8, Misses: 2, Bypass: 5, HitRate: 0.8}}, nil
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
	cases := []struct{ path, expected string }{
		{"/admin/v1/cache/stats?days=1", `"hit_rate":0.8`},
		{"/admin/v1/cache/stats?days=1", `"cacheable":10`},
		{"/admin/v1/cache/stats?days=1", `"bypass":5`},
		{"/admin/v1/cache/report?days=7", `"by_day":[{"day":"2026-11-16"`},
		{"/admin/v1/cache/report?days=7", `"events":[{"time":"10:44"`},
		{"/admin/v1/cache/events?limit=10", `"status":"hit"`},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		request.Header.Set("Authorization", "Bearer "+key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), tc.expected) {
			t.Fatalf("%s: unexpected response %d %s", tc.path, response.Code, response.Body.String())
		}
	}
}

func TestAdminCacheHitRateIgnoresBypass(t *testing.T) {
	stats := admin.CacheStats{Hits: 3, Misses: 1, Bypass: 40}
	if stats.Bypass == 0 {
		t.Fatal("bypass must be tracked separately from cacheable traffic")
	}
	if got := float64(stats.Hits) / float64(stats.Hits+stats.Misses); got != 0.75 {
		t.Fatalf("hit rate over cacheable traffic = %v, want 0.75", got)
	}
}
