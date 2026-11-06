package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
)

type staticRepository struct {
	record auth.KeyRecord
	err    error
}

func (r staticRepository) FindByPrefix(context.Context, string) (auth.KeyRecord, error) {
	return r.record, r.err
}

func TestAuthenticateReturnsPrincipalForValidKey(t *testing.T) {
	const key = "sk-ds-0123456789abcdef"
	hash, err := auth.HashKey(key)
	if err != nil {
		t.Fatalf("hash key: %v", err)
	}
	repository := staticRepository{record: auth.KeyRecord{
		ID:       "key-1",
		TenantID: "tenant-1",
		KeyHash:  hash,
		Scopes:   []string{"chat:write", "models:read"},
	}}
	verifier := auth.NewVerifier(repository)

	principal, err := verifier.Authenticate(context.Background(), "Bearer "+key)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.KeyID != "key-1" || principal.TenantID != "tenant-1" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if !principal.HasScope("chat:write") || principal.HasScope("admin") {
		t.Fatalf("unexpected scopes: %+v", principal.Scopes)
	}
}

func TestAuthenticateRejectsMissingAndMalformedAuthorization(t *testing.T) {
	verifier := auth.NewVerifier(staticRepository{})

	for _, header := range []string{"", "Basic abc", "Bearer", "Bearer short"} {
		if _, err := verifier.Authenticate(context.Background(), header); !errors.Is(err, auth.ErrInvalidAuthorization) {
			t.Fatalf("header %q: expected ErrInvalidAuthorization, got %v", header, err)
		}
	}
}

func TestAuthenticateRejectsRevokedAndExpiredKeys(t *testing.T) {
	const key = "sk-ds-0123456789abcdef"
	hash, err := auth.HashKey(key)
	if err != nil {
		t.Fatalf("hash key: %v", err)
	}
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	cases := []struct {
		name   string
		record auth.KeyRecord
		want   error
	}{
		{
			name:   "revoked",
			record: auth.KeyRecord{ID: "key-1", TenantID: "tenant-1", KeyHash: hash, RevokedAt: &past},
			want:   auth.ErrKeyRevoked,
		},
		{
			name:   "expired",
			record: auth.KeyRecord{ID: "key-1", TenantID: "tenant-1", KeyHash: hash, ExpiresAt: &past},
			want:   auth.ErrKeyExpired,
		},
		{
			name:   "not expired",
			record: auth.KeyRecord{ID: "key-1", TenantID: "tenant-1", KeyHash: hash, ExpiresAt: &future},
			want:   nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verifier := auth.NewVerifier(staticRepository{record: tc.record})
			verifier.SetClock(func() time.Time { return now })
			_, err := verifier.Authenticate(context.Background(), "Bearer "+key)
			if tc.want == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}
