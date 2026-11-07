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

type fakeAdminKeys struct {
	keys    []admin.APIKey
	created admin.CreatedAPIKey
	err     error
}

func (f fakeAdminKeys) List(context.Context, string) ([]admin.APIKey, error) {
	return f.keys, f.err
}

func (f fakeAdminKeys) Create(context.Context, string, admin.CreateAPIKeyInput) (admin.CreatedAPIKey, error) {
	return f.created, f.err
}

func (f fakeAdminKeys) Revoke(context.Context, string, string) (admin.APIKey, error) {
	return f.keys[0], f.err
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
