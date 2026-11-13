package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

type passthroughTranslator struct{}

func (passthroughTranslator) Translate(_ context.Context, text, _ string) (provider.Translation, error) {
	return provider.Translation{Text: text, SourceLanguage: "en"}, nil
}

func TestAgentRouterAdapterUsesRelayContract(t *testing.T) {
	var gotPath, gotKey, gotVersion string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","model":"gpt-5.6-sol","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapterWithTranslator("agentrouter", passthroughTranslator{})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	response, err := adapter.Chat(context.Background(), provider.Account{
		BaseURL: server.URL,
		APIKey:  "relay-secret",
	}, provider.ChatRequest{
		Model:     "gpt-5.6-sol",
		MaxTokens: 8,
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Reply with the single word: ok"},
		},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Content != "ok" {
		t.Fatalf("unexpected content: %q", response.Content)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("expected relay path /v1/messages, got %q", gotPath)
	}
	if gotKey != "relay-secret" {
		t.Fatalf("x-api-key must be forwarded to the relay, got %q", gotKey)
	}
	if gotVersion != "2023-06-01" {
		t.Fatalf("missing anthropic-version, got %q", gotVersion)
	}
	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) == 0 {
		t.Fatalf("payload missing messages: %+v", gotBody)
	}
}
