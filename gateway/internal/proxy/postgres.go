package proxy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository stores proxy candidates and assignments.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL proxy repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Assignment returns the current account proxy, if any.
func (r *PostgresRepository) Assignment(ctx context.Context, tenantID, accountID string) (Endpoint, bool, error) {
	var endpoint Endpoint
	err := r.pool.QueryRow(ctx, `
		SELECT pe.id::text, pe.address, pe.port, pe.country_code, pe.city_name,
		       pe.username_cipher, pe.password_cipher, pe.valid, pe.quarantined_until,
		       pe.score, pxa.sticky_until
		FROM proxy_assignments pxa
		JOIN proxy_endpoints pe ON pe.id = pxa.proxy_endpoint_id AND pe.tenant_id = pxa.tenant_id
		WHERE pxa.tenant_id = $1 AND pxa.provider_account_id = $2::uuid
	`, tenantID, accountID).Scan(
		&endpoint.ID, &endpoint.Address, &endpoint.Port, &endpoint.CountryCode, &endpoint.CityName,
		&endpoint.UsernameCipher, &endpoint.PasswordCipher, &endpoint.Valid,
		&endpoint.QuarantinedUntil, &endpoint.Score, &endpoint.StickyUntil,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Endpoint{}, false, nil
		}
		return Endpoint{}, false, fmt.Errorf("query proxy assignment: %w", err)
	}
	return endpoint, true, nil
}

// Candidates lists healthy endpoints that are not already used by another
// account of the same provider (ADR-024). Cross-provider reuse is allowed.
func (r *PostgresRepository) Candidates(ctx context.Context, tenantID, accountID, excludeID string, limit int) ([]Endpoint, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT pe.id::text, pe.address, pe.port, pe.country_code, pe.city_name,
		       pe.username_cipher, pe.password_cipher, pe.valid, pe.quarantined_until,
		       pe.score,
		       (SELECT count(*) FROM proxy_assignments a
		        WHERE a.tenant_id = pe.tenant_id AND a.proxy_endpoint_id = pe.id)
		FROM proxy_endpoints pe
		JOIN provider_accounts me ON me.id = $2::uuid AND me.tenant_id = $1
		WHERE pe.tenant_id = $1
		  AND pe.valid = true
		  AND (pe.quarantined_until IS NULL OR pe.quarantined_until <= now())
		  AND ($3::text = '' OR pe.id::text <> $3)
		  AND NOT EXISTS (
		      SELECT 1 FROM proxy_poison pp
		      WHERE pp.tenant_id = pe.tenant_id
		        AND pp.provider_id = me.provider_id
		        AND pp.proxy_endpoint_id = pe.id
		        AND pp.banned_until > now()
		  )
		  AND NOT EXISTS (
		      SELECT 1
		      FROM proxy_assignments pxa
		      JOIN provider_accounts other
		        ON other.id = pxa.provider_account_id AND other.tenant_id = pxa.tenant_id
		      WHERE pxa.tenant_id = pe.tenant_id
		        AND pxa.proxy_endpoint_id = pe.id
		        AND other.provider_id = me.provider_id
		        AND pxa.provider_account_id <> me.id
		  )
		ORDER BY pe.score DESC, pe.last_verified_at DESC NULLS LAST, pe.created_at, pe.id
		LIMIT $4
	`, tenantID, accountID, excludeID, limit)
	if err != nil {
		return nil, fmt.Errorf("query proxy candidates: %w", err)
	}
	defer rows.Close()

	endpoints := make([]Endpoint, 0)
	for rows.Next() {
		var endpoint Endpoint
		if err := rows.Scan(
			&endpoint.ID, &endpoint.Address, &endpoint.Port, &endpoint.CountryCode, &endpoint.CityName,
			&endpoint.UsernameCipher, &endpoint.PasswordCipher, &endpoint.Valid,
			&endpoint.QuarantinedUntil, &endpoint.Score, &endpoint.AssignedCount,
		); err != nil {
			return nil, fmt.Errorf("scan proxy candidate: %w", err)
		}
		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy candidates: %w", err)
	}
	return endpoints, nil
}

// Assign attaches an endpoint to an account and renews the sticky lease. The
// provider id is derived from the account so the per-provider uniqueness holds.
func (r *PostgresRepository) Assign(ctx context.Context, tenantID, accountID, endpointID string) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO proxy_assignments (tenant_id, provider_id, provider_account_id, proxy_endpoint_id, sticky_until)
		SELECT $1, pa.provider_id, pa.id, $3::uuid, now() + interval '30 minutes'
		FROM provider_accounts pa
		WHERE pa.tenant_id = $1 AND pa.id = $2::uuid
		ON CONFLICT (tenant_id, provider_account_id)
		DO UPDATE SET provider_id = excluded.provider_id,
		              proxy_endpoint_id = excluded.proxy_endpoint_id,
		              sticky_until = excluded.sticky_until,
		              updated_at = now()
	`, tenantID, accountID, endpointID)
	if err != nil {
		return fmt.Errorf("assign proxy endpoint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assign proxy endpoint: account %s not found", accountID)
	}
	return nil
}

// RecordFailure lowers the endpoint score and marks the failure time.
func (r *PostgresRepository) RecordFailure(ctx context.Context, tenantID, endpointID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE proxy_endpoints
		SET score = GREATEST(score - 20, 0), last_failure_at = now(),
		    quarantined_until = GREATEST(coalesce(quarantined_until, now()), now() + interval '5 minutes')
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, endpointID)
	if err != nil {
		return fmt.Errorf("record proxy failure: %w", err)
	}
	return nil
}

// RecordSuccess raises the endpoint score within the 0..100 range.
func (r *PostgresRepository) RecordSuccess(ctx context.Context, tenantID, endpointID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE proxy_endpoints
		SET score = LEAST(score + 5, 100), quarantined_until = NULL
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, endpointID)
	if err != nil {
		return fmt.Errorf("record proxy success: %w", err)
	}
	return nil
}

// RecordBan persists a provider-scoped poison (ADR-024). The same endpoint may
// still be used by other providers.
func (r *PostgresRepository) RecordBan(ctx context.Context, tenantID, accountID, endpointID, reason string, until time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO proxy_poison (tenant_id, provider_id, proxy_endpoint_id, banned_until, reason)
		SELECT $1, pa.provider_id, $3::uuid, $4, $5
		FROM provider_accounts pa
		WHERE pa.tenant_id = $1 AND pa.id = $2::uuid
		ON CONFLICT (tenant_id, provider_id, proxy_endpoint_id)
		DO UPDATE SET banned_until = GREATEST(proxy_poison.banned_until, excluded.banned_until),
		              reason = excluded.reason,
		              updated_at = now()
	`, tenantID, accountID, endpointID, until, reason)
	if err != nil {
		return fmt.Errorf("record proxy ban: %w", err)
	}
	return nil
}
