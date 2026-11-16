package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func multipartMessage() provider.ChatMessage {
	return provider.ChatMessage{
		Role: "user",
		Parts: []provider.ContentPart{
			{Type: provider.ContentPartText, Text: "describe"},
			{Type: provider.ContentPartImageURL, ImageURL: &provider.ImageURL{URL: "data:image/png;base64,AAAA"}},
			{Type: provider.ContentPartFile, File: &provider.FileData{Filename: "spec.pdf", FileData: "data:application/pdf;base64,CCCC"}},
			{Type: provider.ContentPartInputAudio, InputAudio: &provider.InputAudio{Data: "BBBB", Format: "wav"}},
		},
	}
}

func TestOpenAIAdapterForwardsMultipartContent(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"c1","model":"gpt-4.1","choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("openai")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	_, err = adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{multipartMessage()},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	messages := received["messages"].([]any)
	content, ok := messages[0].(map[string]any)["content"].([]any)
	if !ok {
		t.Fatalf("openai adapter must forward content parts, got %#v", messages[0])
	}
	if len(content) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(content))
	}
	if content[1].(map[string]any)["type"] != "image_url" {
		t.Fatalf("unexpected image part: %#v", content[1])
	}
}

func TestAnthropicAdapterLowersMultipartContent(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m1","model":"claude-sonnet-4","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("anthropic")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	_, err = adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "be brief"},
			multipartMessage(),
		},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if received["system"] != "be brief" {
		t.Fatalf("unexpected system: %#v", received["system"])
	}

	messages := received["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	if len(content) != 4 {
		t.Fatalf("expected 4 blocks after degradation, got %d: %#v", len(content), content)
	}
	image := content[1].(map[string]any)
	if image["type"] != "image" {
		t.Fatalf("expected image block, got %#v", image)
	}
	source := image["source"].(map[string]any)
	if source["type"] != "base64" || source["media_type"] != "image/png" || source["data"] != "AAAA" {
		t.Fatalf("unexpected image source: %#v", source)
	}
	document := content[2].(map[string]any)
	if document["type"] != "document" || document["source"].(map[string]any)["media_type"] != "application/pdf" {
		t.Fatalf("expected document block, got %#v", document)
	}
	audio := content[3].(map[string]any)
	if audio["type"] != "text" {
		t.Fatalf("audio must degrade to text, got %#v", audio)
	}
}

func TestAgentRouterTranslatesTextPartsAndKeepsMedia(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m1","model":"claude-sonnet-4","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	adapter := newAgentRouterTestAdapter(t, &fakeTranslator{
		forward: map[string]string{"olá": "hello"},
		reverse: map[string]string{"ok": "olá"},
		source:  "pt",
	})
	_, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []provider.ChatMessage{{
			Role: "user",
			Parts: []provider.ContentPart{
				{Type: provider.ContentPartText, Text: "olá"},
				{Type: provider.ContentPartImageURL, ImageURL: &provider.ImageURL{URL: "data:image/png;base64,AAAA"}},
			},
		}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	messages := received["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["text"] != "hello" {
		t.Fatalf("text part not translated: %#v", content[0])
	}
	if content[1].(map[string]any)["type"] != "image" {
		t.Fatalf("image part dropped: %#v", content[1])
	}
}
