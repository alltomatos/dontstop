package proxy

import (
	"context"
	"errors"
	"fmt"

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

// Candidates lists healthy endpoints with less than three assigned accounts.
func (r *PostgresRepository) Candidates(ctx context.Context, tenantID, excludeID string, limit int) ([]Endpoint, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT pe.id::text, pe.address, pe.port, pe.country_code, pe.city_name,
		       pe.username_cipher, pe.password_cipher, pe.valid, pe.quarantined_until,
		       pe.score, count(pxa.id)
		FROM proxy_endpoints pe
		LEFT JOIN proxy_assignments pxa
		  ON pxa.tenant_id = pe.tenant_id AND pxa.proxy_endpoint_id = pe.id
		WHERE pe.tenant_id = $1
		  AND pe.valid = true
		  AND (pe.quarantined_until IS NULL OR pe.quarantined_until <= now())
		  AND ($2::text = '' OR pe.id::text <> $2)
		GROUP BY pe.id
		HAVING count(pxa.id) < 3
		ORDER BY pe.score DESC, pe.last_verified_at DESC NULLS LAST, pe.created_at, pe.id
		LIMIT $3
	`, tenantID, excludeID, limit)
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

// Assign attaches an endpoint to an account and renews the sticky lease.
func (r *PostgresRepository) Assign(ctx context.Context, tenantID, accountID, endpointID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO proxy_assignments (tenant_id, provider_account_id, proxy_endpoint_id, sticky_until)
		VALUES ($1, $2::uuid, $3::uuid, now() + interval '30 minutes')
		ON CONFLICT (tenant_id, provider_account_id)
		DO UPDATE SET proxy_endpoint_id = excluded.proxy_endpoint_id,
		              sticky_until = excluded.sticky_until,
		              updated_at = now()
	`, tenantID, accountID, endpointID)
	if err != nil {
		return fmt.Errorf("assign proxy endpoint: %w", err)
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
