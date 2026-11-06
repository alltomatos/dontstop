package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const PrefixLength = 12

var (
	ErrInvalidAuthorization = errors.New("invalid authorization header")
	ErrKeyNotFound          = errors.New("API key not found")
	ErrKeyRevoked           = errors.New("API key revoked")
	ErrKeyExpired           = errors.New("API key expired")
	ErrInvalidKey           = errors.New("invalid API key")
)

// KeyRecord is the persisted API key material required for authentication.
type KeyRecord struct {
	ID        string
	TenantID  string
	KeyHash   string
	Scopes    []string
	ExpiresAt *time.Time
	RevokedAt *time.Time
}

// Repository looks up API key metadata by its non-secret prefix.
type Repository interface {
	FindByPrefix(ctx context.Context, prefix string) (KeyRecord, error)
}

// Principal is the authenticated caller context.
type Principal struct {
	KeyID    string
	TenantID string
	Scopes   []string
}

// HasScope reports whether the principal has an exact scope.
func (p Principal) HasScope(scope string) bool {
	for _, granted := range p.Scopes {
		if granted == scope {
			return true
		}
	}
	return false
}

type principalContextKey struct{}

// WithPrincipal attaches a principal to a request context.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext reads the authenticated principal from a context.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

// Verifier authenticates Bearer API keys against a repository.
type Verifier struct {
	repository Repository
	now        func() time.Time
}

// NewVerifier creates an API key verifier.
func NewVerifier(repository Repository) *Verifier {
	return &Verifier{repository: repository, now: time.Now}
}

// SetClock overrides the clock for deterministic tests.
func (v *Verifier) SetClock(now func() time.Time) {
	v.now = now
}

// Authenticate validates a Bearer header and returns the caller principal.
func (v *Verifier) Authenticate(ctx context.Context, authorization string) (Principal, error) {
	scheme, key, ok := strings.Cut(strings.TrimSpace(authorization), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(key) == "" {
		return Principal{}, ErrInvalidAuthorization
	}
	key = strings.TrimSpace(key)
	if len(key) < PrefixLength {
		return Principal{}, ErrInvalidAuthorization
	}

	record, err := v.repository.FindByPrefix(ctx, key[:PrefixLength])
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return Principal{}, ErrInvalidKey
		}
		return Principal{}, fmt.Errorf("lookup API key: %w", err)
	}
	if record.RevokedAt != nil {
		return Principal{}, ErrKeyRevoked
	}
	if record.ExpiresAt != nil && !v.now().Before(*record.ExpiresAt) {
		return Principal{}, ErrKeyExpired
	}
	if !VerifyKey(record.KeyHash, key) {
		return Principal{}, ErrInvalidKey
	}

	return Principal{
		KeyID:    record.ID,
		TenantID: record.TenantID,
		Scopes:   append([]string(nil), record.Scopes...),
	}, nil
}
