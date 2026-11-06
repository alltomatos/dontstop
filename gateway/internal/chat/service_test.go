package chat_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type fakeRouter struct{ candidate routing.Candidate }

func (r fakeRouter) Route(ctx context.Context, _, _ string, attempt routing.Attempt) (routing.Candidate, error) {
	if _, err := attempt(ctx, r.candidate); err != nil {
		return r.candidate, err
	}
	return r.candidate, nil
}

type fakeDecrypter struct{}

func (fakeDecrypter) Decrypt([]byte) ([]byte, error) { return []byte("provider-key"), nil }

type fakeAdapter struct {
	response provider.ChatResponse
	chunks   []provider.StreamChunk
}

func (a fakeAdapter) Chat(context.Context, provider.Account, provider.ChatRequest) (provider.ChatResponse, error) {
	return a.response, nil
}

func (a fakeAdapter) ChatStream(context.Context, provider.Account, provider.ChatRequest) (provider.Stream, error) {
	return &fakeStream{chunks: a.chunks}, nil
}

type fakeStream struct {
	chunks []provider.StreamChunk
	index  int
}

func (s *fakeStream) Recv(context.Context) (provider.StreamChunk, error) {
	if s.index >= len(s.chunks) {
		return provider.StreamChunk{}, io.EOF
	}
	chunk := s.chunks[s.index]
	s.index++
	return chunk, nil
}

func (s *fakeStream) Close() error { return nil }

type fakeStore struct {
	value provider.ChatResponse
	found bool
}

func (s *fakeStore) Get(context.Context, string) (provider.ChatResponse, bool, error) {
	return s.value, s.found, nil
}

func (s *fakeStore) Set(context.Context, string, provider.ChatResponse) error { return nil }

type fakeExactCache struct {
	entry cache.Entry
	found bool
	set   bool
}

func (c *fakeExactCache) Get(context.Context, string) (cache.Entry, bool, error) {
	return c.entry, c.found, nil
}

func (c *fakeExactCache) Set(context.Context, string, cache.Meta, cache.Entry, time.Duration) error {
	c.set = true
	return nil
}

func (c *fakeExactCache) Invalidate(context.Context, string, string, string) (int64, error) {
	return 0, nil
}

func newService(adapter provider.Adapter, store chat.IdempotencyStore, exactCache cache.ExactCache) *chat.Service {
	return chat.NewService(
		fakeRouter{candidate: routing.Candidate{AccountID: "account-1", ProviderKind: "openai", BaseURL: "https://provider.example", CredentialCipher: []byte("cipher")}},
		fakeDecrypter{},
		func(string) (provider.Adapter, error) { return adapter, nil },
		store,
		exactCache,
		"v1",
	)
}

func TestChatReturnsProviderResponse(t *testing.T) {
	service := newService(fakeAdapter{response: provider.ChatResponse{ID: "chat-1", Model: "gpt-4.1", Content: "Olá"}}, nil, nil)
	response, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Content != "Olá" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestChatUsesIdempotencyStore(t *testing.T) {
	store := &fakeStore{found: true, value: provider.ChatResponse{ID: "cached", Content: "reused"}}
	service := newService(fakeAdapter{}, store, nil)
	response, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "idem-1")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.ID != "cached" || response.Content != "reused" {
		t.Fatalf("expected cached response, got %+v", response)
	}
}

func TestChatUsesExactCacheHit(t *testing.T) {
	exact := &fakeExactCache{found: true, entry: cache.Entry{Response: provider.ChatResponse{ID: "cached", Content: "hit"}}}
	service := newService(fakeAdapter{response: provider.ChatResponse{ID: "provider", Content: "miss"}}, nil, exact)
	response, status, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.ID != "cached" || status != "hit" {
		t.Fatalf("unexpected result: %+v status=%s", response, status)
	}
}

func TestChatStoresExactCacheOnMiss(t *testing.T) {
	exact := &fakeExactCache{}
	service := newService(fakeAdapter{response: provider.ChatResponse{ID: "provider", Content: "miss"}}, nil, exact)
	response, status, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.ID != "provider" || status != "miss" || !exact.set {
		t.Fatalf("unexpected result: %+v status=%s set=%v", response, status, exact.set)
	}
}

type errorExactCache struct{}

func (errorExactCache) Get(context.Context, string) (cache.Entry, bool, error) {
	return cache.Entry{}, false, context.DeadlineExceeded
}
func (errorExactCache) Set(context.Context, string, cache.Meta, cache.Entry, time.Duration) error {
	return context.DeadlineExceeded
}
func (errorExactCache) Invalidate(context.Context, string, string, string) (int64, error) {
	return 0, nil
}

func TestChatBypassesCacheWhenRedisFails(t *testing.T) {
	service := newService(fakeAdapter{response: provider.ChatResponse{ID: "provider", Content: "ok"}}, nil, errorExactCache{})
	response, status, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.ID != "provider" || status != "bypass" {
		t.Fatalf("unexpected result: %+v status=%s", response, status)
	}
}
func TestStreamWritesSSEChunks(t *testing.T) {
	service := newService(fakeAdapter{chunks: []provider.StreamChunk{
		{Content: "Olá"},
		{Content: " mundo", FinishReason: "stop", Usage: &provider.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}},
	}}, nil, nil)
	recorder := httptest.NewRecorder()
	err := service.Stream(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, recorder)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"content":"Olá"`) || !strings.Contains(body, `"content":" mundo"`) || !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("unexpected SSE body: %s", body)
	}
}
