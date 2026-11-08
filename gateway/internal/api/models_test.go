package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

type apiStaticRepository struct {
	record auth.KeyRecord
}

func (r apiStaticRepository) FindByPrefix(context.Context, string) (auth.KeyRecord, error) {
	return r.record, nil
}

type fakeLister struct {
	models []provider.Model
}

func (l fakeLister) ListModels(context.Context, string) ([]provider.Model, error) {
	return l.models, nil
}

func testVerifier(t *testing.T, scopes []string) (*auth.Verifier, string) {
	t.Helper()
	const key = "sk-ds-0123456789abcdef"
	hash, err := auth.HashKey(key)
	if err != nil {
		t.Fatalf("hash key: %v", err)
	}
	return auth.NewVerifier(apiStaticRepository{record: auth.KeyRecord{
		ID: "key-1", TenantID: "tenant-1", KeyHash: hash, Scopes: scopes,
	}}), key
}

func TestModelsRequiresAuthentication(t *testing.T) {
	verifier, _ := testVerifier(t, []string{"models:read"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Models:    fakeLister{},
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestModelsRequiresModelsReadScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"chat:write"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Models:    fakeLister{},
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestModelsReturnsCatalog(t *testing.T) {
	verifier, key := testVerifier(t, []string{"models:read"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Models: fakeLister{models: []provider.Model{
			{ID: "gpt-4.1", DisplayName: "GPT-4.1", OwnedBy: "openai", Origin: "native", Status: "active"},
			{ID: "claude-sonnet-4", DisplayName: "Claude Sonnet 4", OwnedBy: "anthropic", Status: "cooldown"},
		}},
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var body struct {
		Object string `json:"object"`
		Data   []struct {
			ID             string `json:"id"`
			Object         string `json:"object"`
			OwnedBy        string `json:"owned_by"`
			Origin         string `json:"x-dontstop-origin"`
			XDonstopStatus string `json:"x-dontstop-status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Object != "list" || len(body.Data) != 2 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body.Data[0].Origin != "native" {
		t.Fatalf("unexpected model origin: %+v", body.Data[0])
	}
	if body.Data[1].ID != "claude-sonnet-4" || body.Data[1].XDonstopStatus != "cooldown" {
		t.Fatalf("unexpected model: %+v", body.Data[1])
	}
}
