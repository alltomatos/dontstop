package provider

import (
	"errors"
	"time"
)

// ProxyFailureKind classifies why a proxy-backed attempt failed (ADR-024).
type ProxyFailureKind string

const (
	ProxyFailureUnreachable ProxyFailureKind = "proxy_unreachable"
	ProxyFailureAuth        ProxyFailureKind = "proxy_auth"
	ProxyFailureBan         ProxyFailureKind = "upstream_ban"
)

// ProxyPoisonTTL returns the poison duration for a failure class.
func ProxyPoisonTTL(kind ProxyFailureKind) time.Duration {
	switch kind {
	case ProxyFailureUnreachable:
		return 15 * time.Minute
	case ProxyFailureAuth:
		return time.Hour
	case ProxyFailureBan:
		return 24 * time.Hour
	default:
		return 0
	}
}

// ClassifyProxyFailure reports whether an error should rotate the proxy and, if
// so, which poison class applies. Credential (401/403), quota, rate-limit and
// 5xx errors never rotate the proxy: they are handled by account failover.
func ClassifyProxyFailure(err error) (ProxyFailureKind, bool) {
	switch {
	case errors.Is(err, ErrProxyFailure):
		return ProxyFailureUnreachable, true
	case errors.Is(err, ErrProxyAuth):
		return ProxyFailureAuth, true
	case errors.Is(err, ErrProxyBanned):
		return ProxyFailureBan, true
	default:
		return "", false
	}
}
