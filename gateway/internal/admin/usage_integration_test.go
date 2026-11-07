package admin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestUsageServiceAggregatesCostAndObservability(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	const tenantID = "00000000-0000-0000-0000-000000000000"
	var providerID string
	if err := pool.QueryRow(ctx, `INSERT INTO providers (tenant_id, name, kind, origin, enabled) VALUES ($1, 'openai', 'openai', '9router', true) RETURNING id::text`, tenantID).Scan(&providerID); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO usage_daily (tenant_id, day, provider_id, model, requests, tokens_in, tokens_out, cost_usd, errors) VALUES ($1, current_date, $2::uuid, 'gpt-4.1', 2, 100, 50, 1.25, 0)`, tenantID, providerID); err != nil {
		t.Fatalf("insert usage: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO requests (tenant_id, provider_id, model, status, latency_ms, tokens_in, tokens_out, cost_usd, cache_status) VALUES ($1, $2::uuid, 'gpt-4.1', 'success', 20, 50, 25, 0.5, 'hit')`, tenantID, providerID); err != nil {
		t.Fatalf("insert request: %v", err)
	}
	service := admin.NewUsageService(pool)
	report, err := service.CostReport(ctx, tenantID, 30)
	if err != nil {
		t.Fatalf("cost report: %v", err)
	}
	if report.Summary.CostUSD != 1.25 || len(report.ByProvider) != 1 || len(report.ByDay) != 1 {
		t.Fatalf("unexpected cost report: %+v", report)
	}
	summary, err := service.Observability(ctx, tenantID, 1)
	if err != nil {
		t.Fatalf("observability: %v", err)
	}
	if summary.Requests != 1 || summary.P95MS != 20 {
		t.Fatalf("unexpected observability: %+v", summary)
	}
	cacheStats, err := admin.NewCacheService(pool).Stats(ctx, tenantID, 1)
	if err != nil {
		t.Fatalf("cache stats: %v", err)
	}
	if cacheStats.Hits != 1 || cacheStats.Requests != 1 {
		t.Fatalf("unexpected cache stats: %+v", cacheStats)
	}
}
