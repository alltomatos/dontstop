package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is the payload-free usage record for one completed request.
type Event struct {
	RequestID   string
	TenantID    string
	APIKeyID    string
	ProviderID  string
	AccountID   string
	Model       string
	Status      string
	ErrorCode   string
	CacheStatus string
	TokensIn    int
	TokensOut   int
	LatencyMS   int
	CreatedAt   time.Time
}

// Pricer resolves per-million-token prices.
type Pricer interface {
	Pricing(ctx context.Context, tenantID, providerID, model string) (provider.Pricing, error)
}

// Recorder persists usage, daily aggregates and the outbox transactionally.
type Recorder struct {
	pool   *pgxpool.Pool
	pricer Pricer
}

// NewRecorder creates a usage recorder.
func NewRecorder(pool *pgxpool.Pool, pricer Pricer) *Recorder {
	return &Recorder{pool: pool, pricer: pricer}
}

// Record persists exactly one request, aggregate update and outbox event.
func (r *Recorder) Record(ctx context.Context, event Event) error {
	if event.TenantID == "" || event.Model == "" || event.RequestID == "" {
		return fmt.Errorf("tenant, model and request_id are required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	pricing := provider.Pricing{}
	if event.ProviderID != "" && r.pricer != nil {
		resolved, err := r.pricer.Pricing(ctx, event.TenantID, event.ProviderID, event.Model)
		if err != nil {
			return fmt.Errorf("resolve pricing: %w", err)
		}
		pricing = resolved
	}
	cost := (float64(event.TokensIn)*pricing.InputPerMillion + float64(event.TokensOut)*pricing.OutputPerMillion) / 1_000_000

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin usage transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var insertedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO requests (
			request_id, tenant_id, api_key_id, provider_id, account_id, model, status,
			error_code, tokens_in, tokens_out, cost_usd, latency_ms, cache_status, created_at
		) VALUES (
			$1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid,
			$6, $7, NULLIF($8, ''), $9, $10, $11, $12, NULLIF($13, ''), $14
		)
		ON CONFLICT (request_id) DO NOTHING
		RETURNING id::text
	`, event.RequestID, event.TenantID, event.APIKeyID, event.ProviderID, event.AccountID,
		event.Model, event.Status, event.ErrorCode, event.TokensIn, event.TokensOut,
		cost, event.LatencyMS, event.CacheStatus, event.CreatedAt).Scan(&insertedID)
	if err == pgx.ErrNoRows {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("insert request usage: %w", err)
	}

	if event.ProviderID != "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO usage_daily (tenant_id, day, provider_id, model, requests, tokens_in, tokens_out, cost_usd, errors)
			VALUES ($1, $2::date, $3, $4, 1, $5, $6, $7, $8)
			ON CONFLICT (tenant_id, day, provider_id, model) DO UPDATE SET
				requests = usage_daily.requests + 1,
				tokens_in = usage_daily.tokens_in + EXCLUDED.tokens_in,
				tokens_out = usage_daily.tokens_out + EXCLUDED.tokens_out,
				cost_usd = usage_daily.cost_usd + EXCLUDED.cost_usd,
				errors = usage_daily.errors + EXCLUDED.errors
		`, event.TenantID, event.CreatedAt, event.ProviderID, event.Model, event.TokensIn, event.TokensOut, cost, errorCount(event.Status))
		if err != nil {
			return fmt.Errorf("upsert daily usage: %w", err)
		}
	}

	payload, err := json.Marshal(map[string]any{
		"request_id":   event.RequestID,
		"tenant_id":    event.TenantID,
		"provider_id":  event.ProviderID,
		"account_id":   event.AccountID,
		"model":        event.Model,
		"status":       event.Status,
		"cache_status": event.CacheStatus,
		"tokens_in":    event.TokensIn,
		"tokens_out":   event.TokensOut,
		"cost_usd":     cost,
		"latency_ms":   event.LatencyMS,
		"created_at":   event.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("encode usage event: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO usage_outbox (event_id, payload) VALUES (gen_random_uuid(), $1)`, payload); err != nil {
		return fmt.Errorf("insert usage outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit usage transaction: %w", err)
	}
	return nil
}

func errorCount(status string) int {
	if status == "success" {
		return 0
	}
	return 1
}
