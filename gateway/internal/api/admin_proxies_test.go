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

type fakeAdminProxy struct {
	assignErr   error
	unassignErr error
}

func (f fakeAdminProxy) List(context.Context, string) ([]admin.ProxyEndpoint, error) {
	return nil, nil
}

func (f fakeAdminProxy) Create(context.Context, string, admin.CreateProxyEndpointInput) (admin.ProxyEndpoint, error) {
	return admin.ProxyEndpoint{}, nil
}

func (f fakeAdminProxy) Assign(context.Context, string, string, string, string) error {
	return f.assignErr
}

func (f fakeAdminProxy) Unassign(context.Context, string, string, string) error {
	return f.unassignErr
}

func proxyAssignRequest(t *testing.T, body string, assignErr error) *httptest.ResponseRecorder {
	t.Helper()
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness:  health.NewRegistry(0),
		Verifier:   verifier,
		AdminProxy: fakeAdminProxy{assignErr: assignErr},
	})
	request := httptest.NewRequest(http.MethodPut, "/admin/v1/providers/provider-1/accounts/account-1/proxy", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestAdminProxyAssignRequiresManageScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"chat:write"})
	router := api.NewRouter(api.Dependencies{
		Readiness:  health.NewRegistry(0),
		Verifier:   verifier,
		AdminProxy: fakeAdminProxy{},
	})
	request := httptest.NewRequest(http.MethodPut, "/admin/v1/providers/provider-1/accounts/account-1/proxy", strings.NewReader(`{"proxy_endpoint_id":"x"}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without manage scope, got %d", response.Code)
	}
}

func TestAdminProxyAssignMapsNotFound(t *testing.T) {
	response := proxyAssignRequest(t, `{"proxy_endpoint_id":"00000000-0000-0000-0000-000000000000"}`, admin.ErrNotFound)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAdminProxyAssignMapsLimit(t *testing.T) {
	response := proxyAssignRequest(t, `{"proxy_endpoint_id":"00000000-0000-0000-0000-000000000000"}`, admin.ErrProxyLimit)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", response.Code, response.Body.String())
	}
}
