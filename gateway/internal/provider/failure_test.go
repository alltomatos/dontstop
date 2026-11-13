package provider_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func TestClassifyProxyFailure(t *testing.T) {
	cases := []struct {
		err  error
		kind provider.ProxyFailureKind
		ok   bool
		ttl  time.Duration
	}{
		{provider.ErrProxyFailure, provider.ProxyFailureUnreachable, true, 15 * time.Minute},
		{provider.ErrProxyAuth, provider.ProxyFailureAuth, true, time.Hour},
		{provider.ErrProxyBanned, provider.ProxyFailureBan, true, 24 * time.Hour},
		{provider.ErrProviderAuth, "", false, 0},
		{provider.ErrProviderRateLimited, "", false, 0},
		{provider.ErrProviderQuota, "", false, 0},
		{provider.ErrProviderUnavailable, "", false, 0},
		{errors.New("other"), "", false, 0},
	}
	for _, test := range cases {
		kind, ok := provider.ClassifyProxyFailure(test.err)
		if ok != test.ok || kind != test.kind {
			t.Fatalf("ClassifyProxyFailure(%v) = (%q,%v), want (%q,%v)", test.err, kind, ok, test.kind, test.ok)
		}
		if ok && provider.ProxyPoisonTTL(kind) != test.ttl {
			t.Fatalf("ProxyPoisonTTL(%q) = %v, want %v", kind, provider.ProxyPoisonTTL(kind), test.ttl)
		}
	}
}
