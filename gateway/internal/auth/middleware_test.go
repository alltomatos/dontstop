package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
)

func TestMiddlewareRejectsMissingKeyWithoutLeakingIt(t *testing.T) {
	verifier := auth.NewVerifier(staticRepository{})
	handler := verifier.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without a valid key")
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if response.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("missing WWW-Authenticate header")
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] == nil {
		t.Fatal("expected OpenAI-compatible error body")
	}
}

func TestMiddlewareAttachesPrincipalToContext(t *testing.T) {
	const key = "sk-ds-0123456789abcdef"
	hash, err := auth.HashKey(key)
	if err != nil {
		t.Fatalf("hash key: %v", err)
	}
	repository := staticRepository{record: auth.KeyRecord{
		ID:       "key-1",
		TenantID: "tenant-1",
		KeyHash:  hash,
		Scopes:   []string{"models:read"},
	}}
	handler := auth.NewVerifier(repository).Middleware(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok {
			t.Fatal("principal missing from context")
		}
		if principal.TenantID != "tenant-1" {
			t.Fatalf("unexpected tenant %q", principal.TenantID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
}

var _ = context.Background
