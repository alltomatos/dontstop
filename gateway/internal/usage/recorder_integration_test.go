package usage_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/usage"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestRecorderPersistsUsageAggregatesAndOutboxIdempotently(t *testing.T) {
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
		t.Fatalf("reset: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	const tenant = "00000000-0000-0000-0000-000000000000"
	if _, err := pool.Exec(ctx, "INSERT INTO providers (tenant_id,name,kind,enabled) VALUES ($1,'openai','openai',true)", tenant); err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_models (tenant_id,provider_id,model_id,display_name,active,input_price_per_million,output_price_per_million)
		SELECT $1,id,'gpt-4.1','GPT-4.1',true,1.0,2.0 FROM providers WHERE name='openai'
	`, tenant); err != nil {
		t.Fatalf("model: %v", err)
	}
	var providerID string
	if err := pool.QueryRow(ctx, "SELECT id::text FROM providers WHERE name='openai'").Scan(&providerID); err != nil {
		t.Fatalf("provider id: %v", err)
	}

	recorder := usage.NewRecorder(pool, provider.NewCatalog(pool))
	event := usage.Event{
		RequestID: "req-1", TenantID: tenant, ProviderID: providerID, Model: "gpt-4.1",
		Status: "success", TokensIn: 1000, TokensOut: 500, LatencyMS: 120, CreatedAt: time.Now().UTC(),
	}
	if err := recorder.Record(ctx, event); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := recorder.Record(ctx, event); err != nil {
		t.Fatalf("record duplicate: %v", err)
	}

	var requests int
	var cost float64
	if err := pool.QueryRow(ctx, "SELECT count(*), coalesce(sum(cost_usd),0)::float8 FROM requests").Scan(&requests, &cost); err != nil {
		t.Fatalf("requests: %v", err)
	}
	if requests != 1 || cost < 0.00199 || cost > 0.00201 {
		t.Fatalf("unexpected requests/cost: %d %f", requests, cost)
	}
	var daily int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM usage_daily").Scan(&daily); err != nil {
		t.Fatalf("daily: %v", err)
	}
	if daily != 1 {
		t.Fatalf("expected 1 daily row, got %d", daily)
	}

	var payload string
	if err := pool.QueryRow(ctx, "SELECT payload::text FROM usage_outbox").Scan(&payload); err != nil {
		t.Fatalf("outbox: %v", err)
	}
	if strings.Contains(strings.ToLower(payload), "prompt") || strings.Contains(strings.ToLower(payload), "content") {
		t.Fatalf("usage payload must not contain prompt/response content: %s", payload)
	}
}
