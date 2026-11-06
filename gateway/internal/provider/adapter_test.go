package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func TestOpenAIAdapterTranslatesChatRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization %q", request.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["model"] != "gpt-4.1" || body["stream"] != false {
			t.Fatalf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_1","object":"chat.completion","created":1762387200,"model":"gpt-4.1",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Olá"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}
		}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("openai")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	response, err := adapter.Chat(context.Background(), provider.Account{
		ID: "account-1", BaseURL: server.URL, APIKey: "test-key",
	}, provider.ChatRequest{
		Model: "gpt-4.1",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Oi"},
		},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Content != "Olá" || response.Usage.TotalTokens != 12 || response.FinishReason != "stop" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestAnthropicAdapterTranslatesSystemMessagesAndUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		if request.Header.Get("x-api-key") != "anthropic-key" {
			t.Fatalf("unexpected x-api-key")
		}
		if request.Header.Get("anthropic-version") == "" {
			t.Fatal("missing anthropic-version")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["system"] != "Responda em português." {
			t.Fatalf("unexpected system: %+v", body["system"])
		}
		if body["max_tokens"] != float64(512) {
			t.Fatalf("unexpected max_tokens: %+v", body["max_tokens"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4",
			"content":[{"type":"text","text":"Resposta"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":20,"output_tokens":5}
		}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("anthropic")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	response, err := adapter.Chat(context.Background(), provider.Account{
		ID: "account-1", BaseURL: server.URL, APIKey: "anthropic-key",
	}, provider.ChatRequest{
		Model:     "claude-sonnet-4",
		MaxTokens: 512,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "Responda em português."},
			{Role: "user", Content: "Oi"},
		},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Content != "Resposta" || response.Usage.TotalTokens != 25 || response.FinishReason != "stop" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestAdaptersMapProviderErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("openai")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	_, err = adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "bad"}, provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	})
	if !errors.Is(err, provider.ErrProviderAuth) {
		t.Fatalf("expected ErrProviderAuth, got %v", err)
	}
}
