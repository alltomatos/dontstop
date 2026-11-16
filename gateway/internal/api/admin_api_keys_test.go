package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

type fakeAdminKeys struct {
	keys    []admin.APIKey
	created admin.CreatedAPIKey
	err     error
}

func (f fakeAdminKeys) List(context.Context, string) ([]admin.APIKey, error) {
	return f.keys, f.err
}

func (f fakeAdminKeys) Create(context.Context, string, string, admin.CreateAPIKeyInput) (admin.CreatedAPIKey, error) {
	return f.created, f.err
}

func (f fakeAdminKeys) Revoke(context.Context, string, string, string) (admin.APIKey, error) {
	return f.keys[0], f.err
}

func (f fakeAdminKeys) Get(context.Context, string, string) (admin.APIKey, error) {
	if len(f.keys) == 0 {
		return admin.APIKey{}, admin.ErrNotFound
	}
	return f.keys[0], f.err
}

func (f fakeAdminKeys) Update(context.Context, string, string, string, admin.UpdateAPIKeyInput) (admin.APIKey, error) {
	if len(f.keys) == 0 {
		return admin.APIKey{}, admin.ErrNotFound
	}
	return f.keys[0], f.err
}

func (f fakeAdminKeys) Rotate(context.Context, string, string, string) (admin.CreatedAPIKey, error) {
	return f.created, f.err
}

func (f fakeAdminKeys) Scopes() []string {
	return []string{"chat:write", "models:read"}
}
func TestAdminAPIKeysRequireManageScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"chat:write"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		AdminKeys: fakeAdminKeys{},
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/api-keys", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestAdminAPIKeysCreateAndList(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminKeys{
		keys:    []admin.APIKey{{ID: "key-1", Name: "prod", Prefix: "sk-ds-12345", Scopes: []string{"chat:write"}}},
		created: admin.CreatedAPIKey{APIKey: admin.APIKey{ID: "key-2", Name: "new", Prefix: "sk-ds-abcdef", Scopes: []string{"chat:write"}}, Key: "sk-ds-abcdef-secret"},
	}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		AdminKeys: service,
	})

	createRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/api-keys", strings.NewReader(`{"name":"new","scopes":["chat:write"]}`))
	createRequest.Header.Set("Authorization", "Bearer "+key)
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}
	if !strings.Contains(createResponse.Body.String(), "sk-ds-abcdef-secret") {
		t.Fatal("created secret missing from response")
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/api-keys", nil)
	listRequest.Header.Set("Authorization", "Bearer "+key)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "prod") {
		t.Fatalf("unexpected list: %d %s", listResponse.Code, listResponse.Body.String())
	}
}

func TestAdminAPIKeyLifecycleRoutesAndSecretRedaction(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminKeys{
		keys: []admin.APIKey{{
			ID: "key-1", Name: "prod", Prefix: "sk-ds-12345", Scopes: []string{"chat:write"},
			Status: "active", IPAllowlist: []string{}, CreatedAt: time.Now(),
		}},
		created: admin.CreatedAPIKey{APIKey: admin.APIKey{
			ID: "key-1", Name: "prod", Prefix: "sk-ds-abcdef", Scopes: []string{"chat:write"},
			Status: "active", IPAllowlist: []string{}, CreatedAt: time.Now(),
		}, Key: "sk-ds-abcdef-secret"},
	}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		AdminKeys: service,
	})

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/api-keys", nil)
	listRequest.Header.Set("Authorization", "Bearer "+key)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || strings.Contains(listResponse.Body.String(), "sk-ds-abcdef-secret") {
		t.Fatalf("list leaked or failed: %d %s", listResponse.Code, listResponse.Body.String())
	}

	scopesRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/api-keys/scopes", nil)
	scopesRequest.Header.Set("Authorization", "Bearer "+key)
	scopesResponse := httptest.NewRecorder()
	router.ServeHTTP(scopesResponse, scopesRequest)
	if scopesResponse.Code != http.StatusOK || !strings.Contains(scopesResponse.Body.String(), "chat:write") {
		t.Fatalf("scopes failed: %d %s", scopesResponse.Code, scopesResponse.Body.String())
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/api-keys/key-1", nil)
	getRequest.Header.Set("Authorization", "Bearer "+key)
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), "prod") {
		t.Fatalf("get failed: %d %s", getResponse.Code, getResponse.Body.String())
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/admin/v1/api-keys/key-1", strings.NewReader(`{"name":"prod","scopes":["chat:write"],"expires_at":null,"ip_allowlist":[]}`))
	updateRequest.Header.Set("Authorization", "Bearer "+key)
	updateResponse := httptest.NewRecorder()
	router.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update failed: %d %s", updateResponse.Code, updateResponse.Body.String())
	}

	rotateRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/api-keys/key-1/rotate", nil)
	rotateRequest.Header.Set("Authorization", "Bearer "+key)
	rotateResponse := httptest.NewRecorder()
	router.ServeHTTP(rotateResponse, rotateRequest)
	if rotateResponse.Code != http.StatusOK || !strings.Contains(rotateResponse.Body.String(), "sk-ds-abcdef-secret") {
		t.Fatalf("rotate failed: %d %s", rotateResponse.Code, rotateResponse.Body.String())
	}

	revokeRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/api-keys/key-1/revoke", nil)
	revokeRequest.Header.Set("Authorization", "Bearer "+key)
	revokeResponse := httptest.NewRecorder()
	router.ServeHTTP(revokeResponse, revokeRequest)
	if revokeResponse.Code != http.StatusOK {
		t.Fatalf("revoke failed: %d %s", revokeResponse.Code, revokeResponse.Body.String())
	}
}
