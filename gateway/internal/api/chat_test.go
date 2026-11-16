package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type fakeChatService struct {
	response    provider.ChatResponse
	err         error
	calls       int
	cacheStatus string
	request     provider.ChatRequest
}

func (s *fakeChatService) Chat(_ context.Context, _ string, request provider.ChatRequest, _ string) (provider.ChatResponse, string, error) {
	s.calls++
	s.request = request
	status := s.cacheStatus
	if status == "" {
		status = "bypass"
	}
	return s.response, status, s.err
}

func (s *fakeChatService) Stream(_ context.Context, _ string, request provider.ChatRequest, writer http.ResponseWriter) error {
	s.calls++
	s.request = request
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("data: {\"choices\":[]}\n\ndata: [DONE]\n\n"))
	return s.err
}

func chatRouter(t *testing.T, service *fakeChatService, scopes []string) (http.Handler, string) {
	t.Helper()
	verifier, key := testVerifier(t, scopes)
	return api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Chat:      service,
	}), key
}

func TestChatRequiresChatWriteScope(t *testing.T) {
	service := &fakeChatService{}
	router, key := chatRouter(t, service, []string{"models:read"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
	if service.calls != 0 {
		t.Fatal("service must not be called without scope")
	}
}

func TestChatRejectsInvalidBody(t *testing.T) {
	service := &fakeChatService{}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestChatRejectsOversizedBody(t *testing.T) {
	service := &fakeChatService{}
	verifier, key := testVerifier(t, []string{"chat:write"})
	router := api.NewRouter(api.Dependencies{
		Readiness:   health.NewRegistry(0),
		Verifier:    verifier,
		Chat:        service,
		MaxChatBody: 64,
	})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":"` + strings.Repeat("a", 128) + `"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Error.Type != "request_too_large" {
		t.Fatalf("unexpected error type: %+v", payload)
	}
}

func TestChatAcceptsMultipartContent(t *testing.T) {
	service := &fakeChatService{response: provider.ChatResponse{ID: "chat-1", Model: "gpt-4.1", Content: "ok", FinishReason: "stop"}}
	router, key := chatRouter(t, service, []string{"chat:write"})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":[
		{"type":"text","text":"describe"},
		{"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA"}}
	]}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.calls != 1 {
		t.Fatalf("expected service call, got %d", service.calls)
	}
	message := service.request.Messages[0]
	if len(message.Parts) != 2 || message.Parts[1].Modality() != provider.ModalityImage {
		t.Fatalf("multipart content not decoded: %+v", message)
	}
}

func TestChatRejectsUnknownContentPart(t *testing.T) {
	service := &fakeChatService{}
	router, key := chatRouter(t, service, []string{"chat:write"})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":[{"type":"hologram"}]}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
	if service.calls != 0 {
		t.Fatal("service must not be called for invalid content parts")
	}
}
func TestChatMapsTranslationError(t *testing.T) {
	service := &fakeChatService{err: provider.ErrTranslation}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", response.Code)
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Type != "translation_error" {
		t.Fatalf("unexpected error type: %+v", body)
	}
}

func TestChatMapsUpstreamQuotaError(t *testing.T) {
	service := &fakeChatService{err: fmt.Errorf("%w: %w", routing.ErrAllCandidatesFailed, provider.ErrProviderQuota)}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Type != "billing_error" {
		t.Fatalf("unexpected error type: %+v", body)
	}
}
func TestChatMapsTerminalProviderRequestError(t *testing.T) {
	service := &fakeChatService{err: provider.ErrProviderInvalidRequest}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestChatMapsRequiredProxyError(t *testing.T) {
	service := &fakeChatService{err: fmt.Errorf("%w: %w", routing.ErrAllCandidatesFailed, provider.ErrProxyRequired)}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Type != "proxy_unavailable" {
		t.Fatalf("unexpected error type: %+v", body)
	}
}

func TestChatReturnsJSONResponse(t *testing.T) {
	service := &fakeChatService{response: provider.ChatResponse{
		ID: "chat-1", Model: "gpt-4.1", Content: "Olá", FinishReason: "stop",
		Usage: provider.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Object  string `json:"object"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Object != "chat.completion" || body.Choices[0].Message.Content != "Olá" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestChatSetsCacheHeader(t *testing.T) {
	service := &fakeChatService{
		response:    provider.ChatResponse{ID: "chat-1", Model: "gpt-4.1", Content: "cache", FinishReason: "stop"},
		cacheStatus: "hit",
	}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Header().Get("X-Donstop-Cache") != "hit" {
		t.Fatalf("expected cache header hit, got %q", response.Header().Get("X-Donstop-Cache"))
	}
}
func TestChatStreamsSSE(t *testing.T) {
	service := &fakeChatService{}
	router, key := chatRouter(t, service, []string{"chat:write"})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4.1","stream":true,"messages":[{"role":"user","content":"Oi"}]}`))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "data: [DONE]") {
		t.Fatalf("expected SSE terminator, got %q", response.Body.String())
	}
}

func TestChatReturnsToolCalls(t *testing.T) {
	service := &fakeChatService{response: provider.ChatResponse{
		ID: "chat-1", Model: "gpt-4.1", FinishReason: "tool_calls",
		ToolCalls: []provider.ToolCall{{
			ID: "call_1", Type: "function",
			Function: provider.ToolCallFunction{Name: "ping", Arguments: `{"x":1}`},
		}},
	}}
	router, key := chatRouter(t, service, []string{"chat:write"})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":"ping"}],"tools":[{"type":"function","function":{"name":"ping","parameters":{"type":"object"}}}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if len(service.request.Tools) != 1 || service.request.Tools[0].Function.Name != "ping" {
		t.Fatalf("tools not decoded: %+v", service.request.Tools)
	}
	var payload struct {
		Choices []struct {
			Message struct {
				ToolCalls []provider.ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Choices[0].Message.ToolCalls) != 1 || payload.Choices[0].Message.ToolCalls[0].ID != "call_1" {
		t.Fatalf("tool calls missing from response: %+v", payload)
	}
	if payload.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("unexpected finish reason: %s", payload.Choices[0].FinishReason)
	}
}

func TestChatRejectsToolMessageWithoutID(t *testing.T) {
	service := &fakeChatService{}
	router, key := chatRouter(t, service, []string{"chat:write"})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":"ping"},{"role":"tool","content":"result"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if service.calls != 0 {
		t.Fatal("service must not be called for an invalid tool message")
	}
}

func TestChatRejectsInvalidToolChoice(t *testing.T) {
	service := &fakeChatService{}
	router, key := chatRouter(t, service, []string{"chat:write"})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":"ping"}],"tool_choice":"always"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestChatAcceptsManyTools(t *testing.T) {
	service := &fakeChatService{response: provider.ChatResponse{ID: "chat-1", Model: "gpt-4.1", Content: "ok", FinishReason: "stop"}}
	router, key := chatRouter(t, service, []string{"chat:write"})
	tools := make([]map[string]any, 0, 70)
	for index := 0; index < 70; index++ {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":       fmt.Sprintf("tool_%d", index),
				"parameters": map[string]any{"type": "object"},
			},
		})
	}
	payload, err := json.Marshal(map[string]any{
		"model":    "gpt-4.1",
		"messages": []map[string]any{{"role": "user", "content": "ping"}},
		"tools":    tools,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(payload)))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if len(service.request.Tools) != 70 {
		t.Fatalf("expected 70 tools, got %d", len(service.request.Tools))
	}
}
