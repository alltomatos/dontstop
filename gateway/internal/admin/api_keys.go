package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKey is the admin representation of an API key. The secret is never
// returned here.
type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// CreatedAPIKey is returned once, with the full secret.
type CreatedAPIKey struct {
	APIKey
	Key string `json:"key"`
}

// CreateAPIKeyInput creates an API key.
type CreateAPIKeyInput struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyService is the PostgreSQL-backed admin API key service.
type APIKeyService struct {
	pool *pgxpool.Pool
}

// NewAPIKeyService creates an API key service.
func NewAPIKeyService(pool *pgxpool.Pool) *APIKeyService {
	return &APIKeyService{pool: pool}
}

// List returns API keys for a tenant.
func (s *APIKeyService) List(ctx context.Context, tenantID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT k.id::text, k.name, k.key_prefix, k.scopes::text, k.expires_at, k.revoked_at, k.created_at,
		       (SELECT max(r.created_at) FROM requests r WHERE r.api_key_id = k.id AND r.tenant_id = k.tenant_id)
		FROM api_keys k
		WHERE k.tenant_id = $1
		ORDER BY k.created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query admin API keys: %w", err)
	}
	defer rows.Close()

	keys := make([]APIKey, 0)
	for rows.Next() {
		var key APIKey
		var scopesJSON string
		if err := rows.Scan(&key.ID, &key.Name, &key.Prefix, &scopesJSON, &key.ExpiresAt, &key.RevokedAt, &key.CreatedAt, &key.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan admin API key: %w", err)
		}
		if err := json.Unmarshal([]byte(scopesJSON), &key.Scopes); err != nil {
			return nil, fmt.Errorf("decode API key scopes: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin API keys: %w", err)
	}
	return keys, nil
}

// Create generates, hashes and persists a new API key.
func (s *APIKeyService) Create(ctx context.Context, tenantID string, input CreateAPIKeyInput) (CreatedAPIKey, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreatedAPIKey{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	scopes := make([]string, 0, len(input.Scopes))
	for _, scope := range input.Scopes {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	if len(scopes) == 0 {
		return CreatedAPIKey{}, fmt.Errorf("%w: at least one scope is required", ErrInvalidInput)
	}

	key, err := generateAPIKey()
	if err != nil {
		return CreatedAPIKey{}, err
	}
	hash, err := auth.HashKey(key)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("hash API key: %w", err)
	}
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("encode API key scopes: %w", err)
	}

	var result APIKey
	err = s.pool.QueryRow(ctx, `
		INSERT INTO api_keys (tenant_id, name, key_prefix, key_hash, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		RETURNING id::text, name, key_prefix, scopes::text, expires_at, revoked_at, created_at
	`, tenantID, input.Name, key[:auth.PrefixLength], hash, string(scopesJSON), input.ExpiresAt).Scan(
		&result.ID, &result.Name, &result.Prefix, &scopesJSON, &result.ExpiresAt, &result.RevokedAt, &result.CreatedAt,
	)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("insert API key: %w", err)
	}
	if err := json.Unmarshal([]byte(scopesJSON), &result.Scopes); err != nil {
		return CreatedAPIKey{}, fmt.Errorf("decode created API key scopes: %w", err)
	}
	return CreatedAPIKey{APIKey: result, Key: key}, nil
}

// Revoke marks an API key as revoked.
func (s *APIKeyService) Revoke(ctx context.Context, tenantID, keyID string) (APIKey, error) {
	var result APIKey
	var scopesJSON string
	err := s.pool.QueryRow(ctx, `
		UPDATE api_keys
		SET revoked_at = now()
		WHERE tenant_id = $1 AND id = $2::uuid AND revoked_at IS NULL
		RETURNING id::text, name, key_prefix, scopes::text, expires_at, revoked_at, created_at
	`, tenantID, keyID).Scan(&result.ID, &result.Name, &result.Prefix, &scopesJSON, &result.ExpiresAt, &result.RevokedAt, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return APIKey{}, ErrNotFound
	}
	if err != nil {
		return APIKey{}, fmt.Errorf("revoke API key: %w", err)
	}
	if err := json.Unmarshal([]byte(scopesJSON), &result.Scopes); err != nil {
		return APIKey{}, fmt.Errorf("decode revoked API key scopes: %w", err)
	}
	return result, nil
}

func generateAPIKey() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate API key: %w", err)
	}
	return "sk-ds-" + hex.EncodeToString(buffer), nil
}
