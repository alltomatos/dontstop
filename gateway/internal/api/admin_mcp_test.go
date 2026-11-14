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

func (f fakeAdminMCP) UpdateSettings(context.Context, string, string, *bool, *string) (admin.MCPStatus, error) {
	return f.status, nil
}

func (f fakeAdminMCP) UpdateTool(context.Context, string, string, string, bool) (admin.MCPStatus, error) {
	return f.status, nil
}

func (f fakeAdminMCP) Restart(context.Context, string, string) (admin.MCPStatus, error) {
	return f.status, nil
}

func (f fakeAdminMCP) Activity(context.Context, string, int) (admin.MCPActivity, error) {
	return admin.MCPActivity{TotalCalls: 0, TopTools: []admin.MCPToolCount{}}, nil
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

	restartRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/mcp/restart", nil)
	restartRequest.Header.Set("Authorization", "Bearer "+key)
	restartResponse := httptest.NewRecorder()
	router.ServeHTTP(restartResponse, restartRequest)
	if restartResponse.Code != http.StatusOK {
		t.Fatalf("expected restart 200, got %d: %s", restartResponse.Code, restartResponse.Body.String())
	}

	activityRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/mcp/activity?hours=24", nil)
	activityRequest.Header.Set("Authorization", "Bearer "+key)
	activityResponse := httptest.NewRecorder()
	router.ServeHTTP(activityResponse, activityRequest)
	if activityResponse.Code != http.StatusOK {
		t.Fatalf("expected activity 200, got %d: %s", activityResponse.Code, activityResponse.Body.String())
	}

	toolsRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/mcp/tools", nil)
	toolsRequest.Header.Set("Authorization", "Bearer "+key)
	toolsResponse := httptest.NewRecorder()
	router.ServeHTTP(toolsResponse, toolsRequest)
	if toolsResponse.Code != http.StatusOK || !strings.Contains(toolsResponse.Body.String(), "dontstop_chat") {
		t.Fatalf("expected tools 200, got %d: %s", toolsResponse.Code, toolsResponse.Body.String())
	}
}
