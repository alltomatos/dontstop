package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestMigrateCreatesSchemaAndSeedsTenant(t *testing.T) {
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
		t.Fatalf("run migrations: %v", err)
	}

	var tenantName string
	if err := pool.QueryRow(ctx, "SELECT name FROM tenants WHERE id = '00000000-0000-0000-0000-000000000000'").Scan(&tenantName); err != nil {
		t.Fatalf("read default tenant: %v", err)
	}
	if tenantName == "" {
		t.Fatal("default tenant has an empty name")
	}

	var tableCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_name IN ('tenants','api_keys','providers','provider_accounts','requests','usage_daily','usage_outbox','audit_outbox')
	`).Scan(&tableCount); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tableCount != 8 {
		t.Fatalf("expected 8 domain tables, got %d", tableCount)
	}

	var indexCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM pg_indexes
		WHERE schemaname = 'public'
		  AND indexname IN ('api_keys_key_prefix_idx','provider_accounts_routing_idx','requests_tenant_created_idx')
	`).Scan(&indexCount); err != nil {
		t.Fatalf("count indexes: %v", err)
	}
	if indexCount != 3 {
		t.Fatalf("expected 3 hot-path indexes, got %d", indexCount)
	}

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrations must be idempotent: %v", err)
	}
}
