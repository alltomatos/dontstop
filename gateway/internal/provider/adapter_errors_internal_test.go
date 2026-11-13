package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
