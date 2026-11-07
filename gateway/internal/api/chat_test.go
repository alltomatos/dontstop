package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

type fakeChatService struct {
	response    provider.ChatResponse
	err         error
	calls       int
	cacheStatus string
}

func (s *fakeChatService) Chat(context.Context, string, provider.ChatRequest, string) (provider.ChatResponse, string, error) {
	s.calls++
	status := s.cacheStatus
	if status == "" {
		status = "bypass"
	}
	return s.response, status, s.err
}

func (s *fakeChatService) Stream(_ context.Context, _ string, _ provider.ChatRequest, writer http.ResponseWriter) error {
	s.calls++
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
	router, key := chatRouter(t, service, []string{"chat:write"})
	body := `{"model":"gpt-4.1","messages":[{"role":"user","content":"` + strings.Repeat("a", 1<<20) + `"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
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
