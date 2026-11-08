package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

func TestAdminEndpointsCatalogAndPolicy(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness:      health.NewRegistry(0),
		Verifier:       verifier,
		AdminEndpoints: admin.NewEndpointService(),
	})
	for path, expected := range map[string]string{
		"/admin/v1/endpoints":                         `"chat-completions"`,
		"/admin/v1/endpoints/chat-completions/policy": `"auth_required":true`,
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
