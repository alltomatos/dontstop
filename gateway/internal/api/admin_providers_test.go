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
)

type fakeAdminProviders struct {
	providers []admin.Provider
	account   admin.Account
	err       error
}

func (f fakeAdminProviders) List(context.Context, string) ([]admin.Provider, error) {
	return f.providers, f.err
}

func (f fakeAdminProviders) Get(context.Context, string, string) (admin.Provider, error) {
	if f.err != nil {
		return admin.Provider{}, f.err
	}
	return f.providers[0], nil
}

func (f fakeAdminProviders) AddAccount(context.Context, string, string, admin.AccountInput) (admin.Account, error) {
	return f.account, f.err
}

func TestAdminProvidersRequireManageScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"models:read"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     fakeAdminProviders{},
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/providers", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestAdminProvidersListAndAddAccount(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminProviders{
		providers: []admin.Provider{{ID: "provider-1", Name: "OpenAI", Kind: "openai", Origin: "9router", Enabled: true, AccountsCount: 2}},
		account:   admin.Account{ID: "account-1", Name: "prod", Status: "active", Priority: 100, Weight: 1},
	}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     service,
	})

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/providers", nil)
	listRequest.Header.Set("Authorization", "Bearer "+key)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResponse.Code)
	}
	var listBody struct {
		Data []admin.Provider `json:"data"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].Origin != "9router" {
		t.Fatalf("unexpected list: %+v", listBody.Data)
	}

	addRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/accounts", strings.NewReader(`{"name":"prod","credential":"secret","base_url":"https://api.openai.com"}`))
	addRequest.Header.Set("Authorization", "Bearer "+key)
	addRequest.Header.Set("Content-Type", "application/json")
	addResponse := httptest.NewRecorder()
	router.ServeHTTP(addResponse, addRequest)
	if addResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", addResponse.Code, addResponse.Body.String())
	}
}
