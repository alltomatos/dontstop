package provider_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

type fakeMediaFetcher struct {
	mediaType string
	data      []byte
	err       error
	calls     int
	lastURL   string
}

func (f *fakeMediaFetcher) Fetch(_ context.Context, rawURL string) (string, []byte, error) {
	f.calls++
	f.lastURL = rawURL
	return f.mediaType, f.data, f.err
}

func TestInlineRemoteMediaReplacesHTTPImages(t *testing.T) {
	fetcher := &fakeMediaFetcher{mediaType: "image/png", data: []byte("PNG")}
	request := provider.ChatRequest{Model: "m", Messages: []provider.ChatMessage{{
		Role: "user",
		Parts: []provider.ContentPart{
			{Type: provider.ContentPartText, Text: "look"},
			{Type: provider.ContentPartImageURL, ImageURL: &provider.ImageURL{URL: "https://example.com/a.png"}},
		},
	}}}
	inlined, count, err := provider.InlineRemoteMedia(context.Background(), request, fetcher)
	if err != nil {
		t.Fatalf("inline: %v", err)
	}
	if count != 1 || fetcher.lastURL != "https://example.com/a.png" {
		t.Fatalf("unexpected count/url: %d %q", count, fetcher.lastURL)
	}
	url := inlined.Messages[0].Parts[1].ImageURL.URL
	if url != "data:image/png;base64,UE5H" {
		t.Fatalf("unexpected data URL: %q", url)
	}
	if inlined.Messages[0].Content != "look" {
		t.Fatalf("text content changed: %q", inlined.Messages[0].Content)
	}
}

func TestInlineRemoteMediaSkipsDataURLs(t *testing.T) {
	fetcher := &fakeMediaFetcher{mediaType: "image/png", data: []byte("x")}
	request := provider.ChatRequest{Model: "m", Messages: []provider.ChatMessage{{
		Role:  "user",
		Parts: []provider.ContentPart{{Type: provider.ContentPartImageURL, ImageURL: &provider.ImageURL{URL: "data:image/png;base64,AAAA"}}},
	}}}
	_, count, err := provider.InlineRemoteMedia(context.Background(), request, fetcher)
	if err != nil || count != 0 || fetcher.calls != 0 {
		t.Fatalf("data URLs must not be fetched: count=%d calls=%d err=%v", count, fetcher.calls, err)
	}
}

func TestInlineRemoteMediaWrapsFetchErrors(t *testing.T) {
	fetcher := &fakeMediaFetcher{err: errors.New("boom")}
	request := provider.ChatRequest{Model: "m", Messages: []provider.ChatMessage{{
		Role:  "user",
		Parts: []provider.ContentPart{{Type: provider.ContentPartImageURL, ImageURL: &provider.ImageURL{URL: "https://example.com/a.png"}}},
	}}}
	_, _, err := provider.InlineRemoteMedia(context.Background(), request, fetcher)
	if !errors.Is(err, provider.ErrRemoteMedia) {
		t.Fatalf("expected ErrRemoteMedia, got %v", err)
	}
}

func TestBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "::1", "fc00::1", "fe80::1"}
	for _, raw := range blocked {
		if !provider.BlockedIP(net.ParseIP(raw)) {
			t.Fatalf("%s must be blocked", raw)
		}
	}
	allowed := []string{"8.8.8.8", "1.1.1.1", "2001:4860:4860::8888"}
	for _, raw := range allowed {
		if provider.BlockedIP(net.ParseIP(raw)) {
			t.Fatalf("%s must be allowed", raw)
		}
	}
}
