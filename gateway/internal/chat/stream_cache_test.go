package chat_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func streamRequest() provider.ChatRequest {
	return provider.ChatRequest{
		Model:    "gpt-4.1",
		Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}
}

type recordingExactCache struct {
	entry cache.Entry
	found bool
	set   bool
}

func (c *recordingExactCache) Get(context.Context, string) (cache.Entry, bool, error) {
	return c.entry, c.found, nil
}

func (c *recordingExactCache) Set(_ context.Context, _ string, _ cache.Meta, entry cache.Entry, _ time.Duration) error {
	c.set = true
	c.entry = entry
	return nil
}

func (c *recordingExactCache) Invalidate(context.Context, string, string, string) (int64, error) {
	return 0, nil
}

func TestStreamStoresExactCacheOnMiss(t *testing.T) {
	exact := &recordingExactCache{}
	service := newService(fakeAdapter{chunks: []provider.StreamChunk{
		{Content: "Ol"},
		{Content: "a", FinishReason: "stop", Usage: &provider.Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7}},
	}}, nil, exact)

	recorder := httptest.NewRecorder()
	if err := service.Stream(context.Background(), "tenant-1", streamRequest(), recorder); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if recorder.Header().Get("X-Donstop-Cache") != "miss" {
		t.Fatalf("expected miss header, got %q", recorder.Header().Get("X-Donstop-Cache"))
	}
	if !exact.set {
		t.Fatal("expected the streamed response to be cached")
	}
	if exact.entry.Response.Content != "Ola" {
		t.Fatalf("cached content = %q, want %q", exact.entry.Response.Content, "Ola")
	}
}

func TestStreamReplaysExactCacheHit(t *testing.T) {
	exact := &recordingExactCache{
		found: true,
		entry: cache.Entry{Response: provider.ChatResponse{
			ID: "cached", Model: "gpt-4.1", Content: "reused", FinishReason: "stop",
			Usage: provider.Usage{PromptTokens: 9, CompletionTokens: 1, TotalTokens: 10},
		}},
	}
	adapter := &countingAdapter{}
	service := newService(adapter, nil, exact)

	recorder := httptest.NewRecorder()
	if err := service.Stream(context.Background(), "tenant-1", streamRequest(), recorder); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if recorder.Header().Get("X-Donstop-Cache") != "hit" {
		t.Fatalf("expected hit header, got %q", recorder.Header().Get("X-Donstop-Cache"))
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"content":"reused"`) || !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("unexpected replayed body: %s", body)
	}
	if adapter.calls != 0 {
		t.Fatalf("cache hit must not reach the provider, calls=%d", adapter.calls)
	}
}

func TestStreamDoesNotCacheEmptyCompletion(t *testing.T) {
	exact := &recordingExactCache{}
	service := newService(fakeAdapter{chunks: []provider.StreamChunk{
		{FinishReason: "length", Usage: &provider.Usage{PromptTokens: 1, CompletionTokens: 0, TotalTokens: 1}},
	}}, nil, exact)

	recorder := httptest.NewRecorder()
	if err := service.Stream(context.Background(), "tenant-1", streamRequest(), recorder); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if exact.set {
		t.Fatal("an empty completion must not be cached")
	}
}
