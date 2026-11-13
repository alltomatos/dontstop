package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/webshare"
)

type fakeAdminWebshare struct {
	status admin.WebshareSettings
	result admin.WebshareSyncResult
	err    error
}

func (f fakeAdminWebshare) Status(context.Context, string) (admin.WebshareSettings, error) {
	return f.status, f.err
}

func (f fakeAdminWebshare) Test(context.Context, string, string, string, string) (admin.WebshareTestResult, error) {
	return admin.WebshareTestResult{OK: true, PlanID: "plan-1", State: "active", Available: 4}, f.err
}

func (f fakeAdminWebshare) Save(context.Context, string, string, admin.WebshareSaveInput) (admin.WebshareSettings, error) {
	return f.status, f.err
}

func (f fakeAdminWebshare) Sync(context.Context, string, string) (admin.WebshareSyncResult, error) {
	return f.result, f.err
}

func webshareRouter(t *testing.T, service fakeAdminWebshare) (http.Handler, string) {
	t.Helper()
	verifier, key := testVerifier(t, []string{"manage"})
	return api.NewRouter(api.Dependencies{
		Readiness:     health.NewRegistry(0),
		Verifier:      verifier,
		AdminWebshare: service,
	}), key
}

func TestWebshareStatusNeverReturnsAPIKey(t *testing.T) {
	router, key := webshareRouter(t, fakeAdminWebshare{status: admin.WebshareSettings{
		Configured: true, PlanID: "plan-1", KeyVersion: 2, ProxyCount: 4,
	}})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/proxies/webshare", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(strings.ToLower(response.Body.String()), "api_key") || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("status leaked credentials: %s", response.Body.String())
	}
	var body admin.WebshareSettings
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || !body.Configured || body.ProxyCount != 4 {
		t.Fatalf("unexpected status: %+v err=%v", body, err)
	}
}

func TestWebshareTestMapsUpstreamErrors(t *testing.T) {
	router, key := webshareRouter(t, fakeAdminWebshare{err: webshare.ErrUnauthorized})
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/proxies/webshare/test", strings.NewReader(`{"api_key":"bad"}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "webshare_unauthorized") {
		t.Fatalf("unexpected error body: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "bad") {
		t.Fatalf("error leaked the api key: %s", response.Body.String())
	}
}

func TestWebshareSyncConflict(t *testing.T) {
	router, key := webshareRouter(t, fakeAdminWebshare{err: admin.ErrSyncInProgress})
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/proxies/webshare/sync", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", response.Code, response.Body.String())
	}
}

func (f fakeAdminWebshare) RefreshPool(context.Context, string) error { return nil }

func TestAdminWebshareRefresh(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness:     health.NewRegistry(0),
		Verifier:      verifier,
		AdminWebshare: fakeAdminWebshare{},
	})
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/proxies/webshare/refresh", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAdminWebshareRefreshRequiresManageScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"chat:write"})
	router := api.NewRouter(api.Dependencies{
		Readiness:     health.NewRegistry(0),
		Verifier:      verifier,
		AdminWebshare: fakeAdminWebshare{},
	})
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/proxies/webshare/refresh", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without manage scope, got %d", response.Code)
	}
}
