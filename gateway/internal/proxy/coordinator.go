package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	stickyTTL      = 30 * time.Minute
	quarantineTTL  = 5 * time.Minute
	candidateLimit = 50
	refreshTimeout = 20 * time.Second
)

// Endpoint is a proxy assignment candidate loaded from the pool.
type Endpoint struct {
	ID               string
	Address          string
	Port             int
	CountryCode      string
	CityName         string
	UsernameCipher   []byte
	PasswordCipher   []byte
	Valid            bool
	QuarantinedUntil *time.Time
	Score            int
	AssignedCount    int
	StickyUntil      *time.Time
}

// Lease is the proxy selected for one request.
type Lease struct {
	EndpointID string
	ProxyURL   string
	Mode       string
}

// Repository persists the proxy pool and assignments.
type Repository interface {
	Assignment(ctx context.Context, tenantID, accountID string) (Endpoint, bool, error)
	Candidates(ctx context.Context, tenantID, accountID, excludeID string, limit int) ([]Endpoint, error)
	Assign(ctx context.Context, tenantID, accountID, endpointID string) error
	RecordFailure(ctx context.Context, tenantID, endpointID string) error
	RecordSuccess(ctx context.Context, tenantID, endpointID string) error
	RecordBan(ctx context.Context, tenantID, accountID, endpointID, reason string, until time.Time) error
}

// State coordinates transient quarantine in Redis.
type State interface {
	Quarantine(ctx context.Context, tenantID, endpointID string, ttl time.Duration) error
	Quarantined(ctx context.Context, tenantID, endpointID string) (bool, error)
	ClearFailures(ctx context.Context, tenantID, endpointID string) error
}

// Decrypter decrypts proxy credentials.
type Decrypter interface {
	Decrypt(ciphertext []byte) ([]byte, error)
}

// PoolRefresher re-syncs the upstream pool after a failure (e.g. Webshare).
type PoolRefresher interface {
	RefreshPool(ctx context.Context, tenantID string) error
}

// Coordinator selects, rotates and reports on account proxies.
type Coordinator struct {
	repository    Repository
	state         State
	decrypter     Decrypter
	refresher     PoolRefresher
	now           func() time.Time
	rotateCounter metric.Int64Counter
	poisonCounter metric.Int64Counter
	missCounter   metric.Int64Counter
}

// NewCoordinator creates a proxy coordinator.
func NewCoordinator(repository Repository, state State, decrypter Decrypter) *Coordinator {
	meter := otel.Meter("dontstop/gateway/proxy")
	rotate, _ := meter.Int64Counter("proxy.rotate")
	poison, _ := meter.Int64Counter("proxy.poison")
	miss, _ := meter.Int64Counter("proxy.candidate_miss")
	return &Coordinator{
		repository: repository, state: state, decrypter: decrypter, now: time.Now,
		rotateCounter: rotate, poisonCounter: poison, missCounter: miss,
	}
}

// SetPoolRefresher enables the failure-driven upstream pool refresh.
func (c *Coordinator) SetPoolRefresher(refresher PoolRefresher) {
	c.refresher = refresher
}

// Resolve returns the proxy lease for an account. Off never acquires one.
func (c *Coordinator) Resolve(ctx context.Context, tenantID, accountID, mode string) (Lease, error) {
	normalized := provider.NormalizeProxyMode(mode)
	if normalized == provider.ProxyModeOff {
		return Lease{}, nil
	}

	assigned, found, err := c.repository.Assignment(ctx, tenantID, accountID)
	if err != nil {
		return Lease{}, err
	}
	if found && c.usable(ctx, tenantID, assigned) {
		if assigned.StickyUntil == nil || assigned.StickyUntil.Before(c.now()) {
			_ = c.repository.Assign(ctx, tenantID, accountID, assigned.ID)
		}
		return c.lease(assigned)
	}

	endpoint, found, err := c.next(ctx, tenantID, accountID, assigned.ID)
	if err != nil {
		return Lease{}, err
	}
	if !found {
		if provider.RequiresProxy(normalized) {
			return Lease{}, provider.ErrProxyRequired
		}
		return Lease{}, nil
	}
	if err := c.repository.Assign(ctx, tenantID, accountID, endpoint.ID); err != nil {
		return Lease{}, fmt.Errorf("assign proxy: %w", err)
	}
	return c.lease(endpoint)
}

// Rotate quarantines the failed proxy and leases the next eligible one.
func (c *Coordinator) Rotate(ctx context.Context, tenantID, accountID, failedEndpointID, mode string, kind provider.ProxyFailureKind, ttl time.Duration) (Lease, error) {
	normalized := provider.NormalizeProxyMode(mode)
	if normalized == provider.ProxyModeOff {
		return Lease{}, nil
	}
	if failedEndpointID != "" {
		_ = c.repository.RecordFailure(ctx, tenantID, failedEndpointID)
		_ = c.state.Quarantine(ctx, tenantID, failedEndpointID, quarantineTTL)
		if kind != "" && ttl > 0 {
			_ = c.repository.RecordBan(ctx, tenantID, accountID, failedEndpointID, string(kind), c.now().Add(ttl))
			c.poisonCounter.Add(ctx, 1, metric.WithAttributes(
				attribute.String("dontstop.tenant_id", tenantID),
				attribute.String("proxy.failure_kind", string(kind)),
			))
		}
		if c.refresher != nil {
			tenant := tenantID
			go func() {
				refreshCtx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
				defer cancel()
				_ = c.refresher.RefreshPool(refreshCtx, tenant)
			}()
		}
	}
	endpoint, found, err := c.next(ctx, tenantID, accountID, failedEndpointID)
	if err != nil {
		return Lease{}, err
	}
	if !found {
		if provider.RequiresProxy(normalized) {
			return Lease{}, provider.ErrProxyRequired
		}
		return Lease{}, nil
	}
	if err := c.repository.Assign(ctx, tenantID, accountID, endpoint.ID); err != nil {
		return Lease{}, fmt.Errorf("rotate proxy: %w", err)
	}
	c.rotateCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("dontstop.tenant_id", tenantID),
		attribute.String("proxy.failure_kind", string(kind)),
	))
	return c.lease(endpoint)
}

// Success records a healthy proxy use.
func (c *Coordinator) Success(ctx context.Context, tenantID string, lease Lease) {
	if lease.EndpointID == "" {
		return
	}
	_ = c.repository.RecordSuccess(ctx, tenantID, lease.EndpointID)
	_ = c.state.ClearFailures(ctx, tenantID, lease.EndpointID)
}

func (c *Coordinator) usable(ctx context.Context, tenantID string, endpoint Endpoint) bool {
	if !endpoint.Valid {
		return false
	}
	if endpoint.QuarantinedUntil != nil && endpoint.QuarantinedUntil.After(c.now()) {
		return false
	}
	quarantined, err := c.state.Quarantined(ctx, tenantID, endpoint.ID)
	return err != nil || !quarantined
}

func (c *Coordinator) next(ctx context.Context, tenantID, accountID, excludeID string) (Endpoint, bool, error) {
	candidates, err := c.repository.Candidates(ctx, tenantID, accountID, excludeID, candidateLimit)
	if err != nil {
		return Endpoint{}, false, err
	}
	for _, candidate := range candidates {
		if !c.usable(ctx, tenantID, candidate) {
			continue
		}
		return candidate, true, nil
	}
	c.missCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("dontstop.tenant_id", tenantID)))
	return Endpoint{}, false, nil
}

func (c *Coordinator) lease(endpoint Endpoint) (Lease, error) {
	username := ""
	password := ""
	if len(endpoint.UsernameCipher) > 0 {
		plain, err := c.decrypter.Decrypt(endpoint.UsernameCipher)
		if err != nil {
			return Lease{}, fmt.Errorf("decrypt proxy username: %w", err)
		}
		username = string(plain)
	}
	if len(endpoint.PasswordCipher) > 0 {
		plain, err := c.decrypter.Decrypt(endpoint.PasswordCipher)
		if err != nil {
			return Lease{}, fmt.Errorf("decrypt proxy password: %w", err)
		}
		password = string(plain)
	}
	proxyURL, err := provider.BuildProxyURL(endpoint.Address, endpoint.Port, username, password)
	if err != nil {
		return Lease{}, fmt.Errorf("build proxy URL: %w", err)
	}
	return Lease{EndpointID: endpoint.ID, ProxyURL: proxyURL}, nil
}
