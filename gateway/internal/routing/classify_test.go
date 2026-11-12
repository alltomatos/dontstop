package routing_test

import (
	"errors"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

func TestClassifyProviderError(t *testing.T) {
	cases := []struct {
		err  error
		want routing.FailureKind
	}{
		{provider.ErrProviderAuth, routing.FailureAuth},
		{provider.ErrProviderQuota, routing.FailureQuota},
		{provider.ErrProviderRateLimited, routing.FailureRateLimit},
		{provider.ErrProviderUnavailable, routing.FailureServer},
		{provider.ErrProviderResponse, routing.FailureUnknown},
		{provider.ErrProviderInvalidRequest, routing.FailureTerminal},
		{provider.ErrTranslation, routing.FailureTerminal},
		{errors.New("other"), routing.FailureUnknown},
	}
	for _, tc := range cases {
		if got := routing.ClassifyProviderError(tc.err); got != tc.want {
			t.Fatalf("%v: expected %s, got %s", tc.err, tc.want, got)
		}
	}
}
