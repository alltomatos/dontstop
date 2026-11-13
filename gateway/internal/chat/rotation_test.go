package chat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/proxy"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type rotatingResolver struct {
	lease      proxy.Lease
	resolveErr error
	rotate     proxy.Lease
	rotateErr  error
	rotations  int
	successes  int
}

func (r *rotatingResolver) Resolve(context.Context, string, string, string) (proxy.Lease, error) {
	return r.lease, r.resolveErr
}

func (r *rotatingResolver) Rotate(context.Context, string, string, string, string, provider.ProxyFailureKind, time.Duration) (proxy.Lease, error) {
	r.rotations++
	return r.rotate, r.rotateErr
}

func (r *rotatingResolver) Success(context.Context, string, proxy.Lease) {
	r.successes++
}

type sequenceAdapter struct {
	errors  []error
	proxies []string
	calls   int
}

func (a *sequenceAdapter) Chat(_ context.Context, account provider.Account, _ provider.ChatRequest) (provider.ChatResponse, error) {
	a.proxies = append(a.proxies, account.ProxyURL)
	index := a.calls
	a.calls++
	if index < len(a.errors) && a.errors[index] != nil {
		return provider.ChatResponse{}, a.errors[index]
	}
	return provider.ChatResponse{Content: "ok"}, nil
}

func (a *sequenceAdapter) ChatStream(context.Context, provider.Account, provider.ChatRequest) (provider.Stream, error) {
	return &fakeStream{}, nil
}

func proxyService(adapter provider.Adapter, resolver chat.ProxyResolver) *chat.Service {
	service := chat.NewService(
		fakeRouter{candidate: routing.Candidate{
			AccountID:        "account-1",
			ProviderKind:     "openai",
			BaseURL:          "https://provider.example",
			CredentialCipher: []byte("cipher"),
			ProxyMode:        "preferred",
		}},
		fakeDecrypter{},
		func(string) (provider.Adapter, error) { return adapter, nil },
		nil, nil, "v1", nil,
	)
	service.SetProxyResolver(resolver)
	return service
}

func TestChatRotatesProxyBeforeProviderAuth(t *testing.T) {
	adapter := &sequenceAdapter{errors: []error{provider.ErrProxyFailure, nil}}
	resolver := &rotatingResolver{
		lease:  proxy.Lease{EndpointID: "proxy-1", ProxyURL: "http://proxy-1"},
		rotate: proxy.Lease{EndpointID: "proxy-2", ProxyURL: "http://proxy-2"},
	}
	service := proxyService(adapter, resolver)
	if _, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resolver.rotations != 1 || adapter.calls != 2 || resolver.successes != 1 {
		t.Fatalf("unexpected rotation flow: rotations=%d calls=%d successes=%d", resolver.rotations, adapter.calls, resolver.successes)
	}
	if len(adapter.proxies) != 2 || adapter.proxies[0] != "http://proxy-1" || adapter.proxies[1] != "http://proxy-2" {
		t.Fatalf("unexpected proxy sequence: %v", adapter.proxies)
	}
}

func TestChatPreferredFallsBackDirectAfterRotationExhaustion(t *testing.T) {
	adapter := &sequenceAdapter{errors: []error{provider.ErrProxyFailure, nil}}
	resolver := &rotatingResolver{
		lease:  proxy.Lease{EndpointID: "proxy-1", ProxyURL: "http://proxy-1"},
		rotate: proxy.Lease{},
	}
	service := proxyService(adapter, resolver)
	if _, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(adapter.proxies) != 2 || adapter.proxies[0] != "http://proxy-1" || adapter.proxies[1] != "" {
		t.Fatalf("expected direct fallback, got %v", adapter.proxies)
	}
}

func TestChatDoesNotRotateProviderAuthErrors(t *testing.T) {
	adapter := &sequenceAdapter{errors: []error{provider.ErrProviderAuth}}
	resolver := &rotatingResolver{lease: proxy.Lease{EndpointID: "proxy-1", ProxyURL: "http://proxy-1"}}
	service := proxyService(adapter, resolver)
	_, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if !errors.Is(err, provider.ErrProviderAuth) {
		t.Fatalf("expected provider auth error, got %v", err)
	}
	if resolver.rotations != 0 || adapter.calls != 1 {
		t.Fatalf("401/403 must not rotate: rotations=%d calls=%d", resolver.rotations, adapter.calls)
	}
}

func TestChatRequiredProxyFailsBeforeUpstream(t *testing.T) {
	adapter := &sequenceAdapter{}
	resolver := &rotatingResolver{resolveErr: provider.ErrProxyRequired}
	service := proxyService(adapter, resolver)
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

func TestChatRotatesOnUpstreamBan(t *testing.T) {
	adapter := &sequenceAdapter{errors: []error{provider.ErrProxyBanned, nil}}
	resolver := &rotatingResolver{
		lease:  proxy.Lease{EndpointID: "proxy-1", ProxyURL: "http://proxy-1"},
		rotate: proxy.Lease{EndpointID: "proxy-2", ProxyURL: "http://proxy-2"},
	}
	service := proxyService(adapter, resolver)
	if _, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resolver.rotations != 1 || adapter.calls != 2 {
		t.Fatalf("upstream ban must rotate: rotations=%d calls=%d", resolver.rotations, adapter.calls)
	}
}

func TestChatDoesNotRotateRateLimit(t *testing.T) {
	adapter := &sequenceAdapter{errors: []error{provider.ErrProviderRateLimited}}
	resolver := &rotatingResolver{lease: proxy.Lease{EndpointID: "proxy-1", ProxyURL: "http://proxy-1"}}
	service := proxyService(adapter, resolver)
	_, _, err := service.Chat(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}},
	}, "")
	if !errors.Is(err, provider.ErrProviderRateLimited) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if resolver.rotations != 0 || adapter.calls != 1 {
		t.Fatalf("rate limit must not rotate the proxy: rotations=%d calls=%d", resolver.rotations, adapter.calls)
	}
}
