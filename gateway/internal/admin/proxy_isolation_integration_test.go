package admin_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

const proxyIsolationMigration = "0020_provider_proxy_isolation.sql"

func openMigratedPool(t *testing.T, files fs.FS) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, testDatabaseURL(t))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := database.Migrate(ctx, pool, files); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func insertProvider(t *testing.T, pool *pgxpool.Pool, tenantID, name string) string {
	t.Helper()
	var providerID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, $2, 'openai', 'native', true) RETURNING id::text
	`, tenantID, name).Scan(&providerID); err != nil {
		t.Fatalf("insert provider %s: %v", name, err)
	}
	return providerID
}

func insertEndpoint(t *testing.T, pool *pgxpool.Pool, tenantID, address string, port int) string {
	t.Helper()
	var endpointID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO proxy_endpoints (tenant_id, address, port, source)
		VALUES ($1, $2, $3, 'webshare') RETURNING id::text
	`, tenantID, address, port).Scan(&endpointID); err != nil {
		t.Fatalf("insert endpoint %s:%d: %v", address, port, err)
	}
	return endpointID
}

func TestProxyIsolationSchema(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"

	for _, column := range []struct{ table, name string }{
		{"providers", "proxy_auto"},
		{"proxy_assignments", "provider_id"},
		{"proxy_endpoints", "banned_until"},
		{"proxy_endpoints", "ban_reason"},
		{"proxy_endpoints", "last_failure_kind"},
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
			)
		`, column.table, column.name).Scan(&exists); err != nil {
			t.Fatalf("check column %s.%s: %v", column.table, column.name, err)
		}
		if !exists {
			t.Fatalf("missing column %s.%s", column.table, column.name)
		}
	}

	var poisonExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'proxy_poison'
		)
	`).Scan(&poisonExists); err != nil {
		t.Fatalf("check proxy_poison: %v", err)
	}
	if !poisonExists {
		t.Fatal("missing proxy_poison table")
	}

	providerID := insertProvider(t, pool, tenantID, "openai")
	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	providers := admin.NewProviderService(pool, box)
	account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "prod", Credential: "sk-test"})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}

	for _, mode := range []string{"inherit", "off", "manual", "auto"} {
		if _, err := pool.Exec(ctx, `UPDATE provider_accounts SET proxy_mode = $1 WHERE id = $2::uuid`, mode, account.ID); err != nil {
			t.Fatalf("proxy_mode %q must be accepted: %v", mode, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE provider_accounts SET proxy_mode = 'bogus' WHERE id = $2::uuid`, account.ID); err == nil {
		t.Fatal("proxy_mode must reject unknown values")
	}

	var proxyAutoDefault bool
	if err := pool.QueryRow(ctx, `SELECT proxy_auto FROM providers WHERE id = $1::uuid`, providerID).Scan(&proxyAutoDefault); err != nil {
		t.Fatalf("read proxy_auto: %v", err)
	}
	if proxyAutoDefault {
		t.Fatal("proxy_auto must default to false")
	}

	endpointID := insertEndpoint(t, pool, tenantID, "198.51.100.10", 8080)
	if _, err := pool.Exec(ctx, `
		INSERT INTO proxy_assignments (tenant_id, provider_id, provider_account_id, proxy_endpoint_id)
		VALUES ($1, $2::uuid, $3::uuid, $4::uuid)
	`, tenantID, providerID, account.ID, endpointID); err != nil {
		t.Fatalf("insert assignment: %v", err)
	}
	var providerIDValue *string
	if err := pool.QueryRow(ctx, `
		SELECT provider_id::text FROM proxy_assignments WHERE provider_account_id = $1::uuid
	`, account.ID).Scan(&providerIDValue); err != nil {
		t.Fatalf("read assignment provider_id: %v", err)
	}
	if providerIDValue == nil || *providerIDValue != providerID {
		t.Fatalf("assignment provider_id mismatch: %v", providerIDValue)
	}

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migration must be idempotent: %v", err)
	}
}

func TestProxyAssignmentOnePerProviderWithCrossProviderReuse(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"

	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)

	providerA := insertProvider(t, pool, tenantID, "openai")
	providerB := insertProvider(t, pool, tenantID, "anthropic")
	accountA1, err := providers.AddAccount(ctx, tenantID, providerA, admin.AccountInput{Name: "a1", Credential: "sk-a1"})
	if err != nil {
		t.Fatalf("add account a1: %v", err)
	}
	accountA2, err := providers.AddAccount(ctx, tenantID, providerA, admin.AccountInput{Name: "a2", Credential: "sk-a2"})
	if err != nil {
		t.Fatalf("add account a2: %v", err)
	}
	accountB1, err := providers.AddAccount(ctx, tenantID, providerB, admin.AccountInput{Name: "b1", Credential: "sk-b1"})
	if err != nil {
		t.Fatalf("add account b1: %v", err)
	}
	endpoint, err := proxies.Create(ctx, tenantID, admin.CreateProxyEndpointInput{Address: "198.51.100.20", Port: 9000})
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}

	if err := proxies.Assign(ctx, tenantID, providerA, accountA1.ID, endpoint.ID); err != nil {
		t.Fatalf("assign a1: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerA, accountA2.ID, endpoint.ID); !errors.Is(err, admin.ErrProxyLimit) {
		t.Fatalf("same provider must not share an IP: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerB, accountB1.ID, endpoint.ID); err != nil {
		t.Fatalf("cross-provider reuse must be allowed: %v", err)
	}

	var assigned int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM proxy_assignments
		WHERE tenant_id = $1 AND proxy_endpoint_id = $2::uuid
	`, tenantID, endpoint.ID).Scan(&assigned); err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if assigned != 2 {
		t.Fatalf("expected 2 assignments (one per provider), got %d", assigned)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO proxy_poison (tenant_id, provider_id, proxy_endpoint_id, banned_until, reason)
		VALUES ($1, $2::uuid, $3::uuid, now() + interval '15 minutes', 'proxy_unreachable')
	`, tenantID, providerA, endpoint.ID); err != nil {
		t.Fatalf("insert poison: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO proxy_poison (tenant_id, provider_id, proxy_endpoint_id, banned_until, reason)
		VALUES ($1, $2::uuid, $3::uuid, now() + interval '15 minutes', 'proxy_unreachable')
	`, tenantID, providerA, endpoint.ID); err == nil {
		t.Fatal("proxy_poison must be unique per provider and endpoint")
	}
}

func TestProxyAssignmentDedupMigration(t *testing.T) {
	pool := openMigratedPool(t, migrationsBefore(t, proxyIsolationMigration))
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"

	providerID := insertProvider(t, pool, tenantID, "openai")
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	older, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "older", Credential: "sk-older"})
	if err != nil {
		t.Fatalf("add older: %v", err)
	}
	newer, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "newer", Credential: "sk-newer"})
	if err != nil {
		t.Fatalf("add newer: %v", err)
	}
	endpointID := insertEndpoint(t, pool, tenantID, "198.51.100.30", 9100)
	if _, err := pool.Exec(ctx, `
		INSERT INTO proxy_assignments (tenant_id, provider_account_id, proxy_endpoint_id, created_at)
		VALUES ($1, $2::uuid, $4::uuid, now() - interval '1 hour'),
		       ($1, $3::uuid, $4::uuid, now())
	`, tenantID, older.ID, newer.ID, endpointID); err != nil {
		t.Fatalf("seed shared assignments: %v", err)
	}

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("apply isolation migration: %v", err)
	}

	var remaining int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM proxy_assignments WHERE tenant_id = $1 AND proxy_endpoint_id = $2::uuid
	`, tenantID, endpointID).Scan(&remaining); err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected 1 assignment after dedup, got %d", remaining)
	}
	var survivor string
	if err := pool.QueryRow(ctx, `
		SELECT provider_account_id::text FROM proxy_assignments WHERE tenant_id = $1 AND proxy_endpoint_id = $2::uuid
	`, tenantID, endpointID).Scan(&survivor); err != nil {
		t.Fatalf("read survivor: %v", err)
	}
	if survivor != older.ID {
		t.Fatalf("dedup must keep the earliest assignment, got %s", survivor)
	}
}

func migrationsBefore(t *testing.T, skip string) fs.FS {
	t.Helper()
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	for _, name := range names {
		if name >= skip {
			continue
		}
		data, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	return files
}

func TestSetProxyAutoPersists(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	providerID := insertProvider(t, pool, tenantID, "openai")
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)

	enabled, err := providers.SetProxyAuto(ctx, tenantID, providerID, true)
	if err != nil || !enabled {
		t.Fatalf("set proxy_auto: %v %v", enabled, err)
	}
	provider, err := providers.Get(ctx, tenantID, providerID)
	if err != nil || !provider.ProxyAuto {
		t.Fatalf("expected proxy_auto true: %+v err=%v", provider, err)
	}
	if _, err := providers.SetProxyAuto(ctx, tenantID, "not-a-uuid", true); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestProxyListExposesBanPerAccount(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	providerID := insertProvider(t, pool, tenantID, "openai")
	account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "prod", Credential: "sk-test"})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	endpoint, err := proxies.Create(ctx, tenantID, admin.CreateProxyEndpointInput{Address: "198.51.100.50", Port: 9300})
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerID, account.ID, endpoint.ID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO proxy_poison (tenant_id, provider_id, proxy_endpoint_id, banned_until, reason)
		VALUES ($1, $2::uuid, $3::uuid, now() + interval '1 hour', 'upstream_ban')
	`, tenantID, providerID, endpoint.ID); err != nil {
		t.Fatalf("insert poison: %v", err)
	}
	endpoints, err := proxies.List(ctx, tenantID)
	if err != nil || len(endpoints) != 1 || len(endpoints[0].Accounts) != 1 {
		t.Fatalf("unexpected list: %+v err=%v", endpoints, err)
	}
	assigned := endpoints[0].Accounts[0]
	if assigned.BannedUntil == nil || assigned.BanReason != "upstream_ban" {
		t.Fatalf("expected ban metadata, got %+v", assigned)
	}
}

func TestConcurrentAssignSameProviderKeepsSingleAssignment(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	providerID := insertProvider(t, pool, tenantID, "openai")
	endpoint, err := proxies.Create(ctx, tenantID, admin.CreateProxyEndpointInput{Address: "198.51.100.60", Port: 9400})
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	const attempts = 8
	accountIDs := make([]string, attempts)
	for index := range accountIDs {
		account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{
			Name: fmt.Sprintf("acct-%d", index), Credential: "sk-test",
		})
		if err != nil {
			t.Fatalf("add account %d: %v", index, err)
		}
		accountIDs[index] = account.ID
	}

	var wait sync.WaitGroup
	results := make(chan error, attempts)
	for _, accountID := range accountIDs {
		wait.Add(1)
		go func(id string) {
			defer wait.Done()
			results <- proxies.Assign(ctx, tenantID, providerID, id, endpoint.ID)
		}(accountID)
	}
	wait.Wait()
	close(results)

	success, limited := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, admin.ErrProxyLimit):
			limited++
		default:
			t.Fatalf("unexpected assign error: %v", err)
		}
	}
	if success != 1 || limited != attempts-1 {
		t.Fatalf("expected 1 success and %d proxy-limit errors, got %d/%d", attempts-1, success, limited)
	}
	var rows int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM proxy_assignments WHERE tenant_id = $1 AND proxy_endpoint_id = $2::uuid
	`, tenantID, endpoint.ID).Scan(&rows); err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected exactly 1 assignment, got %d", rows)
	}
}

func TestAssignRejectsUnknownEndpointWithoutLeaking(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	providerID := insertProvider(t, pool, tenantID, "openai")
	account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "prod", Credential: "sk-test"})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerID, account.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerID, "not-a-uuid", "not-a-uuid"); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected not found for invalid ids, got %v", err)
	}
}

func TestProviderAccountHealthReflectsPoison(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	providerID := insertProvider(t, pool, tenantID, "openai")
	account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "prod", Credential: "sk-test"})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	endpoint, err := proxies.Create(ctx, tenantID, admin.CreateProxyEndpointInput{Address: "198.51.100.70", Port: 9500})
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerID, account.ID, endpoint.ID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	provider, err := providers.Get(ctx, tenantID, providerID)
	if err != nil || provider.Accounts[0].Proxy == nil || provider.Accounts[0].Proxy.Health != "healthy" {
		t.Fatalf("expected healthy proxy before poison: %+v err=%v", provider.Accounts, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO proxy_poison (tenant_id, provider_id, proxy_endpoint_id, banned_until, reason)
		VALUES ($1, $2::uuid, $3::uuid, now() + interval '1 hour', 'upstream_ban')
	`, tenantID, providerID, endpoint.ID); err != nil {
		t.Fatalf("insert poison: %v", err)
	}
	provider, err = providers.Get(ctx, tenantID, providerID)
	if err != nil || provider.Accounts[0].Proxy == nil || provider.Accounts[0].Proxy.Health != "quarantined" {
		t.Fatalf("expected quarantined health after poison: %+v err=%v", provider.Accounts, err)
	}
}

func TestAssignRejectsSQLiLookingID(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	providerID := insertProvider(t, pool, tenantID, "openai")
	account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "prod", Credential: "sk-test"})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerID, account.ID, "' OR '1'='1"); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected not found for SQLi-looking id, got %v", err)
	}
}

func TestAgentRouterRelayBaseURL(t *testing.T) {
	pool := openMigratedPool(t, migrations.FS)
	ctx := context.Background()
	const tenantID = "00000000-0000-0000-0000-000000000000"
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	providers.SetAgentRouterRelayURL("http://agentrouter-relay:7187")

	provider, err := providers.CreateFromCatalog(ctx, tenantID, admin.CreateFromCatalogInput{
		Slug: "agentrouter", AccountName: "qa", Credential: "sk-test",
	})
	if err != nil {
		t.Fatalf("create from catalog: %v", err)
	}
	got, err := providers.Get(ctx, tenantID, provider.ID)
	if err != nil || len(got.Accounts) == 0 || got.Accounts[0].BaseURL != "http://agentrouter-relay:7187" {
		t.Fatalf("expected relay base url, got %+v err=%v", got.Accounts, err)
	}
	account, err := providers.AddAccount(ctx, tenantID, provider.ID, admin.AccountInput{Name: "second", Credential: "sk-test-2"})
	if err != nil || account.BaseURL != "http://agentrouter-relay:7187" {
		t.Fatalf("expected relay base url on add account, got %+v err=%v", account, err)
	}
}
