package chat

import (
	"context"

	"github.com/alltomatos/dontstop/gateway/internal/proxy"
)

const maxProxyRotations = 2

// ProxyResolver selects and rotates account proxies.
type ProxyResolver interface {
	Resolve(ctx context.Context, tenantID, accountID, mode string) (proxy.Lease, error)
	Rotate(ctx context.Context, tenantID, accountID, failedEndpointID, mode string) (proxy.Lease, error)
	Success(ctx context.Context, tenantID string, lease proxy.Lease)
}

// SetProxyResolver enables automatic proxy selection and rotation.
func (s *Service) SetProxyResolver(resolver ProxyResolver) {
	s.proxies = resolver
}
