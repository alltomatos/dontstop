package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/mcp/catalog"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeStore struct {
	config    admin.MCPConfig
	configErr error
	overrides map[string]bool
	overErr   error
	audits    []string
}

func (f *fakeStore) Settings(context.Context, string) (admin.MCPConfig, error) {
	return f.config, f.configErr
}

func (f *fakeStore) ToolOverrides(context.Context, string) (map[string]bool, error) {
	return f.overrides, f.overErr
}

func (f *fakeStore) RecordAudit(_ context.Context, _, _, event string, _ map[string]any) error {
	f.audits = append(f.audits, event)
	return nil
}

type fakeChat struct {
	request  provider.ChatRequest
	response provider.ChatResponse
	cache    string
	err      error
}

func (f *fakeChat) Chat(_ context.Context, _ string, request provider.ChatRequest, _ string) (provider.ChatResponse, string, error) {
	f.request = request
	return f.response, f.cache, f.err
}

type fakeModels struct {
	models []provider.Model
	err    error
}

func (f fakeModels) ListModels(context.Context, string) ([]provider.Model, error) {
	return f.models, f.err
}

type fakeUsage struct {
	report admin.CostReport
	err    error
}

func (f fakeUsage) CostReport(context.Context, string, int) (admin.CostReport, error) {
	return f.report, f.err
}

type fakeCache struct {
	stats  admin.CacheStats
	series []admin.CacheDay
	err    error
}

func (f fakeCache) Stats(context.Context, string, int) (admin.CacheStats, error) {
	return f.stats, f.err
}

func (f fakeCache) ByDay(context.Context, string, int) ([]admin.CacheDay, error) {
	return f.series, f.err
}

type fakeCombos struct {
	tenantID string
	created  admin.ComboInput
	combo    admin.Combo
	err      error
}

func (f *fakeCombos) List(context.Context, string) ([]admin.Combo, error) {
	return []admin.Combo{f.combo}, f.err
}

func (f *fakeCombos) Get(context.Context, string, string) (admin.Combo, error) {
	if f.err != nil {
		return admin.Combo{}, f.err
	}
	return f.combo, nil
}

func (f *fakeCombos) Create(_ context.Context, tenantID string, input admin.ComboInput) (admin.Combo, error) {
	f.tenantID = tenantID
	f.created = input
	if f.err != nil {
		return admin.Combo{}, f.err
	}
	return f.combo, nil
}

func (f *fakeCombos) Update(context.Context, string, string, int, admin.ComboInput) (admin.Combo, error) {
	return f.combo, f.err
}

func (f *fakeCombos) SetStatus(context.Context, string, string, string) (admin.Combo, error) {
	return f.combo, f.err
}

func (f *fakeCombos) Delete(context.Context, string, string) error {
	return f.err
}

func (f *fakeCombos) Catalog(context.Context, string) ([]admin.ComboCatalogModel, error) {
	return []admin.ComboCatalogModel{{ProviderID: "p1", ModelID: "m1"}}, f.err
}

func TestFilterToolsByScopeAndOverride(t *testing.T) {
	tools := []catalog.Tool{
		{ID: "chat", Scope: "chat:write"},
		{ID: "models", Scope: "models:read"},
		{ID: "health"},
		{ID: "combos", Scope: "combos:write"},
	}
	filtered := filterTools(tools, map[string]bool{"models": false}, []string{"chat:write"})
	if len(filtered) != 2 || filtered[0].ID != "chat" || filtered[1].ID != "health" {
		t.Fatalf("unexpected filtered tools: %+v", filtered)
	}
}

func TestGateRejectsDisabledMismatchedAndUnauthenticated(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	principal := auth.Principal{TenantID: "tenant", Scopes: []string{"chat:write"}}

	for _, test := range []struct {
		name       string
		config     admin.MCPConfig
		withAuth   bool
		transport  string
		wantStatus int
	}{
		{name: "unauthenticated", config: admin.MCPConfig{Enabled: true, Transport: "streamable-http"}, wantStatus: http.StatusUnauthorized},
		{name: "disabled", config: admin.MCPConfig{Enabled: false, Transport: "streamable-http"}, withAuth: true, wantStatus: http.StatusNotFound},
		{name: "transport mismatch", config: admin.MCPConfig{Enabled: true, Transport: "sse"}, withAuth: true, wantStatus: http.StatusNotFound},
		{name: "enabled", config: admin.MCPConfig{Enabled: true, Transport: "streamable-http"}, withAuth: true, transport: "streamable-http", wantStatus: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(Dependencies{Store: &fakeStore{config: test.config}})
			request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if test.withAuth {
				request = request.WithContext(auth.WithPrincipal(request.Context(), principal))
			}
			response := httptest.NewRecorder()
			service.Gate(test.transport, next).ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("expected %d, got %d", test.wantStatus, response.Code)
			}
		})
	}
}

func TestServerExposesOnlyProvidedToolsAndCallsHealth(t *testing.T) {
	principal := auth.Principal{TenantID: "tenant", KeyID: "key", Scopes: []string{"models:read"}}
	service := NewService(Dependencies{})
	health, _ := catalog.Find("dontstop_health")
	session := connectInMemory(t, service.NewServer(principal, []catalog.Tool{health}))

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "dontstop_health" {
		t.Fatalf("unexpected tools: %+v", tools.Tools)
	}
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "dontstop_health"})
	if err != nil {
		t.Fatalf("call health: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("unexpected health result: %+v", result)
	}
	text, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok || text.Text == "" {
		t.Fatalf("expected text content, got %+v", result.Content[0])
	}
}

func TestChatToolUsesTenantAndRecordsAudit(t *testing.T) {
	store := &fakeStore{audits: []string{}}
	chat := &fakeChat{response: provider.ChatResponse{Model: "m", Content: "ok", Usage: provider.Usage{TotalTokens: 3}}, cache: "hit"}
	service := NewService(Dependencies{Store: store, Chat: chat})
	tool, _ := catalog.Find("dontstop_chat")
	session := connectInMemory(t, service.NewServer(auth.Principal{TenantID: "tenant", KeyID: "key", Scopes: []string{"chat:write"}}, []catalog.Tool{tool}))

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "dontstop_chat",
		Arguments: map[string]any{"model": "m", "messages": []any{map[string]any{"role": "user", "content": "hi"}}},
	})
	if err != nil {
		t.Fatalf("call chat: %v", err)
	}
	if result.IsError {
		t.Fatalf("chat returned tool error: %+v", result)
	}
	if chat.request.Model != "m" || len(chat.request.Messages) != 1 {
		t.Fatalf("unexpected chat request: %+v", chat.request)
	}
	if len(store.audits) != 1 || store.audits[0] != "mcp.tool.call" {
		t.Fatalf("unexpected audits: %+v", store.audits)
	}
}

func TestComboCreateToolCallsStoreAndAudits(t *testing.T) {
	store := &fakeStore{audits: []string{}}
	combos := &fakeCombos{combo: admin.Combo{ID: "combo-1", Slug: "qa"}}
	service := NewService(Dependencies{Store: store, Combos: combos})
	tool, _ := catalog.Find("dontstop_combo_create")
	session := connectInMemory(t, service.NewServer(auth.Principal{TenantID: "tenant", KeyID: "key", Scopes: []string{"combos:write"}}, []catalog.Tool{tool}))

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "dontstop_combo_create",
		Arguments: map[string]any{
			"slug": "qa", "name": "QA", "strategy": "priority",
			"members": []any{map[string]any{"provider_id": "00000000-0000-0000-0000-000000000001", "model_id": "m", "enabled": true, "weight": 1}},
		},
	})
	if err != nil {
		t.Fatalf("call combo create: %v", err)
	}
	if result.IsError {
		if len(result.Content) > 0 {
			if text, ok := result.Content[0].(*sdkmcp.TextContent); ok {
				t.Fatalf("combo create returned tool error: %s", text.Text)
			}
		}
		t.Fatalf("combo create returned tool error: %+v", result)
	}
	if combos.created.Slug != "qa" || len(combos.created.Members) != 1 {
		t.Fatalf("unexpected combo input: %+v", combos.created)
	}
	if combos.tenantID != "tenant" {
		t.Fatalf("combo tool used wrong tenant: %q", combos.tenantID)
	}
	if len(store.audits) != 2 || store.audits[0] != "mcp.combo.create" || store.audits[1] != "mcp.tool.call" {
		t.Fatalf("unexpected audits: %+v", store.audits)
	}
}

func TestComboToolReturnsSanitizedNotFound(t *testing.T) {
	combos := &fakeCombos{err: admin.ErrNotFound}
	service := NewService(Dependencies{Store: &fakeStore{}, Combos: combos})
	tool, _ := catalog.Find("dontstop_combo_get")
	session := connectInMemory(t, service.NewServer(auth.Principal{TenantID: "tenant", Scopes: []string{"combos:read"}}, []catalog.Tool{tool}))
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "dontstop_combo_get", Arguments: map[string]any{"id": "missing"}})
	if err != nil {
		t.Fatalf("call combo get: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got %+v", result)
	}
}

func TestUnavailableBackendsReturnToolError(t *testing.T) {
	service := NewService(Dependencies{})
	for _, id := range []string{"dontstop_embeddings", "dontstop_web_search"} {
		tool, _ := catalog.Find(id)
		session := connectInMemory(t, service.NewServer(auth.Principal{TenantID: "tenant", Scopes: []string{"embeddings:write", "web:read"}}, []catalog.Tool{tool}))
		result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: id})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if !result.IsError {
			t.Fatalf("%s: expected tool error", id)
		}
	}
}

func TestComboErrorSanitized(t *testing.T) {
	if err := comboError(admin.ErrNotFound); err == nil || err.Error() != "combo not found" {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := comboError(errors.New("database detail")); err == nil || err.Error() != "combo operation failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStdioServesInitializeAndTools(t *testing.T) {
	service := NewService(Dependencies{Store: &fakeStore{config: admin.MCPConfig{Enabled: true, Transport: "stdio"}}})
	principal := auth.Principal{TenantID: "tenant", KeyID: "key", Scopes: []string{"models:read"}}

	oldStdin, oldStdout := os.Stdin, os.Stdout
	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	os.Stdin, os.Stdout = inReader, outWriter
	t.Cleanup(func() {
		os.Stdin, os.Stdout = oldStdin, oldStdout
		_ = inReader.Close()
		_ = inWriter.Close()
		_ = outReader.Close()
		_ = outWriter.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- service.RunStdio(ctx, principal) }()

	frames := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}` + "\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"
	if _, err := io.WriteString(inWriter, frames); err != nil {
		t.Fatalf("write stdio frames: %v", err)
	}

	reader := bufio.NewReader(outReader)
	seen := map[int]bool{}
	for index := 0; index < 2; index++ {
		lineCh := make(chan string, 1)
		errCh := make(chan error, 1)
		go func() {
			line, err := reader.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			lineCh <- line
		}()
		select {
		case line := <-lineCh:
			var message struct {
				ID     int             `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  any             `json:"error"`
			}
			if err := json.Unmarshal([]byte(line), &message); err != nil {
				t.Fatalf("decode stdio frame: %v (%s)", err, line)
			}
			if message.Error != nil {
				t.Fatalf("stdio error for id %d: %v", message.ID, message.Error)
			}
			if len(message.Result) == 0 {
				t.Fatalf("missing result for id %d", message.ID)
			}
			if message.ID == 2 && !strings.Contains(string(message.Result), "dontstop_models") {
				t.Fatalf("tools/list missing models: %s", message.Result)
			}
			seen[message.ID] = true
		case err := <-errCh:
			t.Fatalf("read stdio response: %v", err)
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for stdio response")
		}
	}
	if !seen[1] || !seen[2] {
		t.Fatalf("missing stdio responses: %+v", seen)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && ctx.Err() == nil {
			t.Fatalf("RunStdio returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunStdio did not stop after context cancellation")
	}
}

func TestRunStdioRejectsDisabledTenant(t *testing.T) {
	service := NewService(Dependencies{Store: &fakeStore{config: admin.MCPConfig{Enabled: false, Transport: "stdio"}}})
	if err := service.RunStdio(context.Background(), auth.Principal{TenantID: "tenant"}); err == nil {
		t.Fatal("expected disabled MCP error")
	}
}

func TestModelsAndUsageTools(t *testing.T) {
	service := NewService(Dependencies{
		Store:  &fakeStore{},
		Models: fakeModels{models: []provider.Model{{ID: "model-1", DisplayName: "Model 1"}}},
		Usage:  fakeUsage{report: admin.CostReport{Summary: admin.CostSummary{Days: 30}}},
		Cache: fakeCache{
			stats:  admin.CacheStats{Days: 7, Hits: 3, Misses: 1, Bypass: 9, Cacheable: 4, HitRate: 0.75},
			series: []admin.CacheDay{{Day: "2026-11-16", Hits: 3, Misses: 1, Bypass: 9, HitRate: 0.75}},
		},
	})
	principal := auth.Principal{TenantID: "tenant", Scopes: []string{"models:read", "usage:read"}}
	modelsTool, _ := catalog.Find("dontstop_models")
	usageTool, _ := catalog.Find("dontstop_usage")
	cacheTool, _ := catalog.Find("dontstop_cache")
	session := connectInMemory(t, service.NewServer(principal, []catalog.Tool{modelsTool, usageTool, cacheTool}))

	modelsResult, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "dontstop_models"})
	if err != nil || modelsResult.IsError {
		t.Fatalf("models tool: err=%v result=%+v", err, modelsResult)
	}
	usageResult, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "dontstop_usage", Arguments: map[string]any{"days": 7}})
	if err != nil || usageResult.IsError {
		t.Fatalf("usage tool: err=%v result=%+v", err, usageResult)
	}
	cacheResult, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "dontstop_cache", Arguments: map[string]any{"days": 7}})
	if err != nil || cacheResult.IsError {
		t.Fatalf("cache tool: err=%v result=%+v", err, cacheResult)
	}
	encoded, err := json.Marshal(cacheResult.StructuredContent)
	if err != nil {
		t.Fatalf("encode cache result: %v", err)
	}
	for _, want := range []string{`"hit_rate":0.75`, `"bypass":9`, `"by_day":[`, `"day":"2026-11-16"`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("cache tool payload missing %s: %s", want, encoded)
		}
	}
}

func TestCacheToolRequiresUsageScope(t *testing.T) {
	cacheTool, ok := catalog.Find("dontstop_cache")
	if !ok {
		t.Fatal("cache tool missing")
	}
	if cacheTool.Scope != "usage:read" {
		t.Fatalf("cache tool scope = %q, want usage:read", cacheTool.Scope)
	}
	service := NewService(Dependencies{Store: &fakeStore{}, Cache: fakeCache{}})
	allowed, err := service.AllowedTools(context.Background(), auth.Principal{TenantID: "tenant", Scopes: []string{"models:read"}})
	if err != nil {
		t.Fatalf("allowed tools: %v", err)
	}
	for _, tool := range allowed {
		if tool.ID == "dontstop_cache" {
			t.Fatal("cache tool must not be announced without usage:read")
		}
	}
}

func connectInMemory(t *testing.T, server *sdkmcp.Server) *sdkmcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}
