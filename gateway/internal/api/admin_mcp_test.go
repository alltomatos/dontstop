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

type fakeAdminMCP struct {
	status admin.MCPStatus
}

func (f fakeAdminMCP) Status(context.Context, string) (admin.MCPStatus, error) {
	return f.status, nil
}

func (f fakeAdminMCP) UpdateSettings(context.Context, string, *bool, *string) (admin.MCPStatus, error) {
	return f.status, nil
}

func (f fakeAdminMCP) UpdateTool(context.Context, string, string, bool) (admin.MCPStatus, error) {
	return f.status, nil
}

func TestAdminMCPStatusAndSettings(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	status := admin.MCPStatus{
		Enabled: true, Transport: "streamable-http",
		Tools: []admin.MCPTool{{ID: "dontstop_chat", Name: "dontstop_chat", Scope: "chat:write", Enabled: true}},
	}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		AdminMCP:  fakeAdminMCP{status: status},
	})

	getRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/mcp/status", nil)
	getRequest.Header.Set("Authorization", "Bearer "+key)
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), "streamable-http") {
		t.Fatalf("unexpected status response: %d %s", getResponse.Code, getResponse.Body.String())
	}

	patchRequest := httptest.NewRequest(http.MethodPatch, "/admin/v1/mcp/settings", strings.NewReader(`{"enabled":true,"transport":"sse"}`))
	patchRequest.Header.Set("Authorization", "Bearer "+key)
	patchResponse := httptest.NewRecorder()
	router.ServeHTTP(patchResponse, patchRequest)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", patchResponse.Code, patchResponse.Body.String())
	}
}
