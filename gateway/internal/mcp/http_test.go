package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStreamableHTTPTransportListsAndCalls(t *testing.T) {
	store := &fakeStore{config: admin.MCPConfig{Enabled: true, Transport: "streamable-http"}}
	service := NewService(Dependencies{Store: store})
	principal := auth.Principal{TenantID: "tenant", KeyID: "key", Scopes: []string{"models:read"}}
	handler := withPrincipal(principal, service.Gate("streamable-http", service.StreamableHandler()))
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint:             server.URL + "/mcp",
		HTTPClient:           server.Client(),
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("connect streamable: %v", err)
	}
	defer func() { _ = session.Close() }()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 3 {
		t.Fatalf("expected models/health/provider_list, got %+v", tools.Tools)
	}
	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "dontstop_health"})
	if err != nil || result.IsError {
		t.Fatalf("call health: err=%v result=%+v", err, result)
	}
}

func TestStreamableHTTPDisabledReturnsNotFound(t *testing.T) {
	service := NewService(Dependencies{Store: &fakeStore{config: admin.MCPConfig{Enabled: false, Transport: "streamable-http"}}})
	handler := withPrincipal(auth.Principal{TenantID: "tenant", Scopes: []string{"models:read"}}, service.Gate("streamable-http", service.StreamableHandler()))
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestLegacySSETransportListsTools(t *testing.T) {
	store := &fakeStore{config: admin.MCPConfig{Enabled: true, Transport: "sse"}}
	service := NewService(Dependencies{Store: store})
	principal := auth.Principal{TenantID: "tenant", KeyID: "key", Scopes: []string{"models:read"}}
	handler := withPrincipal(principal, service.Gate("sse", service.SSEHandler()))
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.SSEClientTransport{
		Endpoint:   server.URL + "/sse",
		HTTPClient: server.Client(),
	}, nil)
	if err != nil {
		t.Fatalf("connect sse: %v", err)
	}
	defer func() { _ = session.Close() }()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list sse tools: %v", err)
	}
	if len(tools.Tools) != 3 {
		t.Fatalf("expected models/health/provider_list, got %+v", tools.Tools)
	}
}

func withPrincipal(principal auth.Principal, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(w, request.WithContext(auth.WithPrincipal(request.Context(), principal)))
	})
}
