package chat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type countingAdapter struct {
	calls    int
	proxyURL string
}

func (a *countingAdapter) Chat(_ context.Context, account provider.Account, _ provider.ChatRequest) (provider.ChatResponse, error) {
	a.calls++
	a.proxyURL = account.ProxyURL
	return provider.ChatResponse{Content: "ok"}, nil
}

func (a *countingAdapter) ChatStream(context.Context, provider.Account, provider.ChatRequest) (provider.Stream, error) {
	a.calls++
	return &fakeStream{}, nil
}

func TestRequiredProxyFailsBeforeUpstream(t *testing.T) {
	adapter := &countingAdapter{}
	service := chat.NewService(
		fakeRouter{candidate: routing.Candidate{
			AccountID:        "account-1",
			ProviderKind:     "openai",
			BaseURL:          "https://provider.example",
			CredentialCipher: []byte("cipher"),
			ProxyMode:        "required",
			ProxyAddress:     "203.0.113.10",
			ProxyPort:        8080,
			ProxyHealthy:     false,
		}},
		fakeDecrypter{},
		func(string) (provider.Adapter, error) { return adapter, nil },
		nil, nil, "v1", nil,
	)
	_, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if !errors.Is(err, provider.ErrProxyRequired) {
		t.Fatalf("expected ErrProxyRequired, got %v", err)
	}
	if adapter.calls != 0 {
		t.Fatalf("upstream must not be called, got %d calls", adapter.calls)
	}
}

func TestOffModeCallsUpstreamDirectly(t *testing.T) {
	adapter := &countingAdapter{}
	service := chat.NewService(
		fakeRouter{candidate: routing.Candidate{
			AccountID:        "account-1",
			ProviderKind:     "openai",
			BaseURL:          "https://provider.example",
			CredentialCipher: []byte("cipher"),
			ProxyMode:        "off",
			ProxyAddress:     "203.0.113.10",
			ProxyPort:        8080,
			ProxyHealthy:     true,
		}},
		fakeDecrypter{},
		func(string) (provider.Adapter, error) { return adapter, nil },
		nil, nil, "v1", nil,
	)
	if _, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if adapter.calls != 1 {
		t.Fatalf("expected one upstream call, got %d", adapter.calls)
	}
	if adapter.proxyURL != "" {
		t.Fatalf("off must connect directly, got proxy %q", adapter.proxyURL)
	}
}
