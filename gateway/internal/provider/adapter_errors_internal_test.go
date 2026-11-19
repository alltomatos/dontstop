package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoProviderRequestMapsNetworkFailureAsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	server.Close()
	_, err := doProviderRequest(context.Background(), server.Client(), server.URL, nil, map[string]string{"probe": "1"}, "application/json", false)
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected unavailable for network failure, got %v", err)
	}
}

func TestDoProviderRequestMapsAgentRouterErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		proxy  bool
		want   error
	}{
		{"budget pool exhausted", http.StatusPaymentRequired, `{"error":{"message":"Budget pool quota has been exhausted"}}`, false, ErrProviderQuota},
		{"no available channel", http.StatusServiceUnavailable, `{"error":{"message":"no available channel"}}`, false, ErrProviderUnavailable},
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`, false, ErrProviderAuth},
		{"rate limited", http.StatusTooManyRequests, `{"error":{"message":"rate limit"}}`, false, ErrProviderRateLimited},
		{"waf ban with proxy", http.StatusForbidden, `{"error":{"message":"your IP address is banned"}}`, true, ErrProxyBanned},
		{"opencode usage limit", http.StatusTooManyRequests, `{"error":{"type":"GoUsageLimitError","message":"Go usage limit exceeded"}}`, false, ErrProviderQuota},
		{"insufficient quota", http.StatusTooManyRequests, `{"error":{"type":"insufficient_quota","message":"You exceeded your current quota"}}`, false, ErrProviderQuota},
		{"billing hard limit", http.StatusTooManyRequests, `{"error":{"message":"billing_hard_limit_reached"}}`, false, ErrProviderQuota},
		{"insufficient account funds", http.StatusPaymentRequired, `Insufficient account funds`, false, ErrProviderQuota},
		{"transient rate limit", http.StatusTooManyRequests, `{"error":{"message":"Rate limit exceeded"}}`, false, ErrProviderRateLimited},
		{"endpoint unavailable", http.StatusTooManyRequests, `{"error":{"code":429,"message":"Upstream request failed: Endpoint is unavailable."}}`, false, ErrProviderRateLimited},
		{"bad gateway stays unavailable", http.StatusBadGateway, `{"error":{"message":"upstream failure"}}`, false, ErrProviderUnavailable},
		{"service unavailable stays unavailable", http.StatusServiceUnavailable, `{"error":{"message":"no available channel"}}`, false, ErrProviderUnavailable},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			_, err := doProviderRequest(context.Background(), server.Client(), server.URL, nil, map[string]string{"probe": "1"}, "application/json", test.proxy)
			if !errors.Is(err, test.want) {
				t.Fatalf("status %d: expected %v, got %v", test.status, test.want, err)
			}
		})
	}
}
