package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

type fakeTranslator struct {
	mu        sync.Mutex
	forward   map[string]string
	reverse   map[string]string
	fail      map[string]error
	source    string
	callCount int
}

func (f *fakeTranslator) Translate(_ context.Context, text, target string) (provider.Translation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	if err := f.fail[text]; err != nil {
		return provider.Translation{}, err
	}
	if target == "en" {
		translated, ok := f.forward[text]
		if !ok {
			return provider.Translation{}, fmt.Errorf("no forward translation for %q", text)
		}
		return provider.Translation{Text: translated, SourceLanguage: f.source}, nil
	}
	translated, ok := f.reverse[text]
	if !ok {
		return provider.Translation{}, fmt.Errorf("no reverse translation for %q", text)
	}
	return provider.Translation{Text: translated, SourceLanguage: "en"}, nil
}

func (f *fakeTranslator) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

func newAgentRouterTestAdapter(t *testing.T, translator provider.Translator) provider.Adapter {
	t.Helper()
	adapter, err := provider.NewAdapterWithTranslator("agentrouter", translator)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return adapter
}

func TestAgentRouterTranslatesPromptBeforeSendingAndResponseBack(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		if request.Header.Get("x-api-key") != "agent-key" {
			t.Fatalf("unexpected x-api-key")
		}
		if request.Header.Get("anthropic-version") == "" {
			t.Fatal("missing anthropic-version")
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4",
			"content":[{"type":"text","text":"Hello from AgentRouter"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":20,"output_tokens":5}
		}`))
	}))
	defer server.Close()

	translator := &fakeTranslator{
		source:  "pt-BR",
		forward: map[string]string{"Você pode ajudar?": "Can you help?", "Responda em português.": "Answer in Portuguese."},
		reverse: map[string]string{"Hello from AgentRouter": "Olá do AgentRouter"},
	}
	adapter := newAgentRouterTestAdapter(t, translator)
	request := provider.ChatRequest{
		Model:     "claude-sonnet-4",
		MaxTokens: 512,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "Responda em português."},
			{Role: "user", Content: "Você pode ajudar?"},
		},
	}

	response, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "agent-key"}, request)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Content != "Olá do AgentRouter" {
		t.Fatalf("expected translated response, got %q", response.Content)
	}
	if request.Messages[1].Content != "Você pode ajudar?" {
		t.Fatal("original prompt must remain unchanged")
	}
	if received["system"] != "Answer in Portuguese." {
		t.Fatalf("system prompt was not translated: %+v", received["system"])
	}
	messages, ok := received["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("unexpected messages: %+v", received["messages"])
	}
	message := messages[0].(map[string]any)
	if message["content"] != "Can you help?" {
		t.Fatalf("user prompt was not translated: %+v", message["content"])
	}
	if response.Usage.TotalTokens != 25 {
		t.Fatalf("unexpected usage: %+v", response.Usage)
	}
}

func TestAgentRouterBlocksRequestWhenPromptTranslationFails(t *testing.T) {
	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called++
	}))
	defer server.Close()

	translator := &fakeTranslator{
		source: "pt-BR",
		fail:   map[string]error{"Você pode ajudar?": errors.New("translator unavailable")},
	}
	adapter := newAgentRouterTestAdapter(t, translator)
	_, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "agent-key"}, provider.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Você pode ajudar?"},
		},
	})
	if !errors.Is(err, provider.ErrTranslation) {
		t.Fatalf("expected ErrTranslation, got %v", err)
	}
	if called != 0 {
		t.Fatal("AgentRouter must not be called when prompt translation fails")
	}
}

func TestAgentRouterBlocksWhenResponseTranslationFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","model":"claude-sonnet-4","content":[{"type":"text","text":"Hello"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	translator := &fakeTranslator{
		source:  "pt-BR",
		forward: map[string]string{"Ol?": "Hello"},
		fail:    map[string]error{"Hello": errors.New("translator unavailable")},
	}
	adapter := newAgentRouterTestAdapter(t, translator)
	_, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "agent-key"}, provider.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []provider.ChatMessage{{Role: "user", Content: "Ol?"}},
	})
	if !errors.Is(err, provider.ErrTranslation) {
		t.Fatalf("expected ErrTranslation, got %v", err)
	}
}

func TestAgentRouterStreamTranslatesBufferedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\n")
		_, _ = io.WriteString(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello \"}}\n\n")
		_, _ = io.WriteString(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"world\"}}\n\n")
		_, _ = io.WriteString(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n")
		_, _ = io.WriteString(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	translator := &fakeTranslator{
		source:  "pt-BR",
		forward: map[string]string{"Olá": "Hello"},
		reverse: map[string]string{"Hello world": "Olá mundo"},
	}
	adapter := newAgentRouterTestAdapter(t, translator)
	stream, err := adapter.ChatStream(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "agent-key"}, provider.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Olá"},
		},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	chunk, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if chunk.Content != "Olá mundo" || chunk.FinishReason != "stop" {
		t.Fatalf("unexpected translated chunk: %+v", chunk)
	}
	if chunk.Usage == nil || chunk.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", chunk.Usage)
	}
	if _, err := stream.Recv(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestAgentRouterStreamBlocksOnResponseTranslationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n")
		_, _ = io.WriteString(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	translator := &fakeTranslator{
		source:  "pt-BR",
		forward: map[string]string{"Olá": "Hello"},
		fail:    map[string]error{"Hello": errors.New("translator unavailable")},
	}
	adapter := newAgentRouterTestAdapter(t, translator)
	stream, err := adapter.ChatStream(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "agent-key"}, provider.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Olá"},
		},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	_, err = stream.Recv(context.Background())
	if !errors.Is(err, provider.ErrTranslation) {
		t.Fatalf("expected ErrTranslation, got %v", err)
	}
}

func TestAgentRouterAcceptsEnglishWithoutReverseTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","model":"claude-sonnet-4","content":[{"type":"text","text":"Hello"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	translator := &fakeTranslator{
		source:  "en",
		forward: map[string]string{"Hello": "Hello"},
	}
	adapter := newAgentRouterTestAdapter(t, translator)
	response, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "agent-key"}, provider.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Content != "Hello" {
		t.Fatalf("unexpected response: %q", response.Content)
	}
	if translator.calls() != 1 {
		t.Fatalf("expected one prompt translation call, got %d", translator.calls())
	}
}
