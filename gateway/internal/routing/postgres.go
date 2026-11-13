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
		       p.name, p.kind, pa.priority, pa.weight, pa.base_url, pa.credential_cipher, pa.credential_key_version,
		       CASE
		           WHEN pa.proxy_mode = 'inherit'
		               THEN CASE WHEN p.proxy_auto THEN 'auto' ELSE 'off' END
		           ELSE pa.proxy_mode
		       END AS proxy_mode,
		       coalesce(pe.id::text, ''), coalesce(pe.address, ''), coalesce(pe.port, 0),
		       coalesce(pe.country_code, ''), coalesce(pe.city_name, ''),
		       pe.username_cipher, pe.password_cipher,
		       coalesce(pe.valid, false)
		           AND (pe.quarantined_until IS NULL OR pe.quarantined_until <= now())
		           AND NOT EXISTS (
		               SELECT 1 FROM proxy_poison pp
		               WHERE pp.tenant_id = pa.tenant_id
		                 AND pp.provider_id = pa.provider_id
		                 AND pp.proxy_endpoint_id = pe.id
		                 AND pp.banned_until > now()
		           )
		FROM provider_accounts pa
		JOIN providers p ON p.id = pa.provider_id AND p.tenant_id = pa.tenant_id
		JOIN provider_models pm ON pm.provider_id = p.id AND pm.tenant_id = pa.tenant_id
		LEFT JOIN proxy_assignments pxa ON pxa.tenant_id = pa.tenant_id AND pxa.provider_account_id = pa.id
		LEFT JOIN proxy_endpoints pe ON pe.id = pxa.proxy_endpoint_id AND pe.tenant_id = pa.tenant_id
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
			&candidate.BaseURL,
			&candidate.CredentialCipher,
			&candidate.CredentialKeyVersion,
			&candidate.ProxyMode,
			&candidate.ProxyEndpointID,
			&candidate.ProxyAddress,
			&candidate.ProxyPort,
			&candidate.ProxyCountry,
			&candidate.ProxyCity,
			&candidate.ProxyUsernameCipher,
			&candidate.ProxyPasswordCipher,
			&candidate.ProxyHealthy,
		); err != nil {
			return nil, fmt.Errorf("scan routing candidate: %w", err)
		}
		candidate.ModelID = model
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routing candidates: %w", err)
	}
	return candidates, nil
}

// CandidatesForTarget lists eligible accounts for one explicit provider/model.
func (r *PostgresRepository) CandidatesForTarget(ctx context.Context, tenantID, providerID, model string) ([]Candidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pa.tenant_id::text, pa.id::text, pa.provider_id::text,
		       p.name, p.kind, pa.priority, pa.weight, pa.base_url, pa.credential_cipher, pa.credential_key_version,
		       CASE
		           WHEN pa.proxy_mode = 'inherit'
		               THEN CASE WHEN p.proxy_auto THEN 'auto' ELSE 'off' END
		           ELSE pa.proxy_mode
		       END AS proxy_mode,
		       coalesce(pe.id::text, ''), coalesce(pe.address, ''), coalesce(pe.port, 0),
		       coalesce(pe.country_code, ''), coalesce(pe.city_name, ''),
		       pe.username_cipher, pe.password_cipher,
		       coalesce(pe.valid, false)
		           AND (pe.quarantined_until IS NULL OR pe.quarantined_until <= now())
		           AND NOT EXISTS (
		               SELECT 1 FROM proxy_poison pp
		               WHERE pp.tenant_id = pa.tenant_id
		                 AND pp.provider_id = pa.provider_id
		                 AND pp.proxy_endpoint_id = pe.id
		                 AND pp.banned_until > now()
		           )
		FROM provider_accounts pa
		JOIN providers p ON p.id = pa.provider_id AND p.tenant_id = pa.tenant_id
		JOIN provider_models pm ON pm.provider_id = p.id AND pm.tenant_id = pa.tenant_id
		LEFT JOIN proxy_assignments pxa ON pxa.tenant_id = pa.tenant_id AND pxa.provider_account_id = pa.id
		LEFT JOIN proxy_endpoints pe ON pe.id = pxa.proxy_endpoint_id AND pe.tenant_id = pa.tenant_id
		WHERE pa.tenant_id = $1
		  AND pa.provider_id = $2::uuid
		  AND pm.model_id = $3
		  AND pm.active = true
		  AND p.enabled = true
		  AND pa.status = 'active'
		  AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now())
		ORDER BY pa.priority, pa.weight DESC, pa.id
	`, tenantID, providerID, model)
	if err != nil {
		return nil, fmt.Errorf("query target candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]Candidate, 0)
	for rows.Next() {
		var candidate Candidate
		if err := rows.Scan(
			&candidate.TenantID, &candidate.AccountID, &candidate.ProviderID,
			&candidate.ProviderName, &candidate.ProviderKind, &candidate.Priority,
			&candidate.Weight, &candidate.BaseURL, &candidate.CredentialCipher,
			&candidate.CredentialKeyVersion, &candidate.ProxyMode,
			&candidate.ProxyEndpointID, &candidate.ProxyAddress, &candidate.ProxyPort,
			&candidate.ProxyCountry, &candidate.ProxyCity, &candidate.ProxyUsernameCipher,
			&candidate.ProxyPasswordCipher, &candidate.ProxyHealthy,
		); err != nil {
			return nil, fmt.Errorf("scan target candidate: %w", err)
		}
		candidate.ModelID = model
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate target candidates: %w", err)
	}
	return candidates, nil
}
