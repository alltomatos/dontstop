package provider

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pricing contains per-million-token prices for a provider model.
type Pricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// Model is a public model entry returned by /v1/models.
type Model struct {
	ID          string
	DisplayName string
	OwnedBy     string
	Status      string
	CreatedAt   int64
}

// Catalog lists models available to a tenant.
type Catalog struct {
	pool *pgxpool.Pool
}

// NewCatalog creates a PostgreSQL-backed model catalog.
func NewCatalog(pool *pgxpool.Pool) *Catalog {
	return &Catalog{pool: pool}
}

// ListModels returns active models from enabled providers with at least one account.
func (c *Catalog) ListModels(ctx context.Context, tenantID string) ([]Model, error) {
	rows, err := c.pool.Query(ctx, `
		SELECT pm.model_id, pm.display_name, p.name, extract(epoch from pm.created_at)::bigint AS created_at,
			CASE WHEN EXISTS (
				SELECT 1 FROM provider_accounts pa
				WHERE pa.provider_id = p.id
				  AND pa.status = 'active'
				  AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now())
			) THEN 'active' ELSE 'cooldown' END AS status
		FROM provider_models pm
		JOIN providers p ON p.id = pm.provider_id AND p.tenant_id = pm.tenant_id
		WHERE pm.tenant_id = $1
		  AND pm.active = true
		  AND p.enabled = true
		  AND EXISTS (SELECT 1 FROM provider_accounts pa WHERE pa.provider_id = p.id)
		ORDER BY pm.model_id
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query models: %w", err)
	}
	defer rows.Close()

	models := make([]Model, 0)
	for rows.Next() {
		var model Model
		if err := rows.Scan(&model.ID, &model.DisplayName, &model.OwnedBy, &model.CreatedAt, &model.Status); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate models: %w", err)
	}
	return models, nil
}

// Pricing returns the per-million-token prices for a provider model.
func (c *Catalog) Pricing(ctx context.Context, tenantID, providerID, model string) (Pricing, error) {
	var pricing Pricing
	err := c.pool.QueryRow(ctx, `
		SELECT input_price_per_million::float8, output_price_per_million::float8
		FROM provider_models
		WHERE tenant_id = $1 AND provider_id = $2 AND model_id = $3
	`, tenantID, providerID, model).Scan(&pricing.InputPerMillion, &pricing.OutputPerMillion)
	if err != nil {
		return Pricing{}, fmt.Errorf("query model pricing: %w", err)
	}
	return pricing, nil
}
