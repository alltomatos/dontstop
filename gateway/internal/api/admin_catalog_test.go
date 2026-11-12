package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

type fakeCatalog struct {
	entries    []admin.CatalogEntry
	provider   admin.Provider
	err        error
	gotSlug    string
	gotCred    string
	gotAccount string
}

func (f *fakeCatalog) ListCatalog(context.Context) ([]admin.CatalogEntry, error) {
	return f.entries, f.err
}

func (f *fakeCatalog) CreateFromCatalog(_ context.Context, _ string, input admin.CreateFromCatalogInput) (admin.Provider, error) {
	f.gotSlug = input.Slug
	f.gotCred = input.Credential
	f.gotAccount = input.AccountName
	if f.err != nil {
		return admin.Provider{}, f.err
	}
	return f.provider, nil
}

func TestAdminCatalogRequiresManageScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"models:read"})
	router := api.NewRouter(api.Dependencies{
		Readiness:    health.NewRegistry(0),
		Verifier:     verifier,
		AdminCatalog: &fakeCatalog{},
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/provider-catalog", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestAdminCatalogList(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		AdminCatalog: &fakeCatalog{entries: []admin.CatalogEntry{
			{Slug: "openrouter", Protocol: "openai", DefaultBaseURL: "https://openrouter.ai/api/v1"},
		}},
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/provider-catalog", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var body struct {
		Object string               `json:"object"`
		Data   []admin.CatalogEntry `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Object != "list" || len(body.Data) != 1 || body.Data[0].Slug != "openrouter" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestAdminCatalogConnectUsesPathSlug(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := &fakeCatalog{provider: admin.Provider{ID: "provider-1", Name: "OpenRouter", Kind: "openai", CatalogSlug: "openrouter"}}
	router := api.NewRouter(api.Dependencies{
		Readiness:    health.NewRegistry(0),
		Verifier:     verifier,
		AdminCatalog: service,
	})
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/provider-catalog/openrouter/connect",
		strings.NewReader(`{"credential":"sk-or-test","slug":"ignored","name":"Hacked Provider","account_name":"openrouter-prod"}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if service.gotSlug != "openrouter" || service.gotCred != "sk-or-test" || service.gotAccount != "openrouter-prod" {
		t.Fatalf("unexpected connect input: slug=%q credential=%q account=%q", service.gotSlug, service.gotCred, service.gotAccount)
	}
}

func TestAdminCatalogConnectErrorMapping(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{err: admin.ErrNotFound, want: http.StatusNotFound},
		{err: admin.ErrInvalidInput, want: http.StatusBadRequest},
		{err: admin.ErrConflict, want: http.StatusConflict},
		{err: fmt.Errorf("boom"), want: http.StatusInternalServerError},
	}
	for _, test := range tests {
		verifier, key := testVerifier(t, []string{"manage"})
		router := api.NewRouter(api.Dependencies{
			Readiness:    health.NewRegistry(0),
			Verifier:     verifier,
			AdminCatalog: &fakeCatalog{err: test.err},
		})
		request := httptest.NewRequest(http.MethodPost, "/admin/v1/provider-catalog/openai/connect",
			strings.NewReader(`{"credential":"sk-test"}`))
		request.Header.Set("Authorization", "Bearer "+key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("error %v: expected %d, got %d", test.err, test.want, response.Code)
		}
	}
}
