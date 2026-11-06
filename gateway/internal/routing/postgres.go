package routing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository loads routing candidates from PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a routing repository backed by pgx.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Candidates lists eligible accounts for a tenant/model in priority order.
func (r *PostgresRepository) Candidates(ctx context.Context, tenantID, model string) ([]Candidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pa.tenant_id::text, pa.id::text, pa.provider_id::text,
		       p.name, p.kind, pa.priority, pa.weight, pa.credential_cipher
		FROM provider_accounts pa
		JOIN providers p ON p.id = pa.provider_id AND p.tenant_id = pa.tenant_id
		JOIN provider_models pm ON pm.provider_id = p.id AND pm.tenant_id = pa.tenant_id
		WHERE pa.tenant_id = $1
		  AND pm.model_id = $2
		  AND pm.active = true
		  AND p.enabled = true
		  AND pa.status = 'active'
		  AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now())
		ORDER BY pa.priority, pa.weight DESC, pa.id
	`, tenantID, model)
	if err != nil {
		return nil, fmt.Errorf("query routing candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]Candidate, 0)
	for rows.Next() {
		var candidate Candidate
		if err := rows.Scan(
			&candidate.TenantID,
			&candidate.AccountID,
			&candidate.ProviderID,
			&candidate.ProviderName,
			&candidate.ProviderKind,
			&candidate.Priority,
			&candidate.Weight,
			&candidate.CredentialCipher,
		); err != nil {
			return nil, fmt.Errorf("scan routing candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routing candidates: %w", err)
	}
	return candidates, nil
}
