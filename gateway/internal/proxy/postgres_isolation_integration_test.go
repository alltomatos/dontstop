package proxy_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/proxy"
	"github.com/alltomatos/dontstop/gateway/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecordBanPoisonsOnlyTheProvider(t *testing.T) {
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
	providerA := insertIsolationProvider(t, pool, tenantID, "provider-a")
	providerB := insertIsolationProvider(t, pool, tenantID, "provider-b")
	accountA := insertIsolationAccount(t, pool, tenantID, providerA, "account-a")
	accountB := insertIsolationAccount(t, pool, tenantID, providerB, "account-b")
	banned := insertIsolationEndpoint(t, pool, tenantID, "198.51.100.40", 9200)
	healthy := insertIsolationEndpoint(t, pool, tenantID, "198.51.100.41", 9201)

	repository := proxy.NewPostgresRepository(pool)
	if err := repository.RecordBan(ctx, tenantID, accountA, banned, "upstream_ban", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("record ban: %v", err)
	}

	candidatesA, err := repository.Candidates(ctx, tenantID, accountA, "", 10)
	if err != nil {
		t.Fatalf("candidates A: %v", err)
	}
	if len(candidatesA) != 1 || candidatesA[0].ID != healthy {
		t.Fatalf("provider A must skip the banned endpoint, got %+v", candidateIDs(candidatesA))
	}
	candidatesB, err := repository.Candidates(ctx, tenantID, accountB, "", 10)
	if err != nil {
		t.Fatalf("candidates B: %v", err)
	}
	if len(candidatesB) != 2 {
		t.Fatalf("provider B must keep both endpoints, got %+v", candidateIDs(candidatesB))
	}

	if _, err := pool.Exec(ctx, `
		UPDATE proxy_poison SET banned_until = now() - interval '1 hour'
		WHERE tenant_id = $1 AND proxy_endpoint_id = $2::uuid
	`, tenantID, banned); err != nil {
		t.Fatalf("expire poison: %v", err)
	}
	candidatesA, err = repository.Candidates(ctx, tenantID, accountA, "", 10)
	if err != nil {
		t.Fatalf("candidates A after expiry: %v", err)
	}
	if len(candidatesA) != 2 {
		t.Fatalf("expired poison must not exclude the endpoint, got %+v", candidateIDs(candidatesA))
	}
}

func insertIsolationProvider(t *testing.T, pool *pgxpool.Pool, tenantID, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, $2, 'openai', 'native', true) RETURNING id::text
	`, tenantID, name).Scan(&id); err != nil {
		t.Fatalf("insert provider %s: %v", name, err)
	}
	return id
}

func insertIsolationAccount(t *testing.T, pool *pgxpool.Pool, tenantID, providerID, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, base_url, credential_cipher, status)
		VALUES ($1, $2::uuid, $3, 'https://example.test', '\x01'::bytea, 'active') RETURNING id::text
	`, tenantID, providerID, name).Scan(&id); err != nil {
		t.Fatalf("insert account %s: %v", name, err)
	}
	return id
}

func insertIsolationEndpoint(t *testing.T, pool *pgxpool.Pool, tenantID, address string, port int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO proxy_endpoints (tenant_id, address, port, source, valid)
		VALUES ($1, $2, $3, 'webshare', true) RETURNING id::text
	`, tenantID, address, port).Scan(&id); err != nil {
		t.Fatalf("insert endpoint %s:%d: %v", address, port, err)
	}
	return id
}

func candidateIDs(endpoints []proxy.Endpoint) []string {
	ids := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		ids = append(ids, endpoint.ID)
	}
	return ids
}
