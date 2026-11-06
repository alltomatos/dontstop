package routing

import (
	"errors"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

// ClassifyProviderError maps provider failures to routing feedback.
func ClassifyProviderError(err error) FailureKind {
	switch {
	case errors.Is(err, provider.ErrProviderAuth):
		return FailureAuth
	case errors.Is(err, provider.ErrProviderRateLimited):
		return FailureRateLimit
	case errors.Is(err, provider.ErrProviderUnavailable):
		return FailureServer
	default:
		return FailureUnknown
	}
}
