package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository loads API key metadata from PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates an API key repository backed by pgx.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// FindByPrefix returns the newest active key for a prefix.
func (r *PostgresRepository) FindByPrefix(ctx context.Context, prefix string) (KeyRecord, error) {
	var record KeyRecord
	var scopesJSON string
	var expiresAt, revokedAt pgtype.Timestamptz

	err := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, key_hash, scopes::text, expires_at, revoked_at
		FROM api_keys
		WHERE key_prefix = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY created_at DESC
		LIMIT 1
	`, prefix).Scan(&record.ID, &record.TenantID, &record.KeyHash, &scopesJSON, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyRecord{}, ErrKeyNotFound
	}
	if err != nil {
		return KeyRecord{}, fmt.Errorf("query API key: %w", err)
	}
	if err := json.Unmarshal([]byte(scopesJSON), &record.Scopes); err != nil {
		return KeyRecord{}, fmt.Errorf("decode API key scopes: %w", err)
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		record.ExpiresAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		record.RevokedAt = &value
	}
	return record, nil
}
