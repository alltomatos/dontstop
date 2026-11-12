package admin_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestProxyModeAndAssignmentPersistWithoutCredentials(t *testing.T) {
	url := testDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	const tenantID = "00000000-0000-0000-0000-000000000000"

	var providerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, 'openai', 'openai', 'native', true) RETURNING id::text
	`, tenantID).Scan(&providerID); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{Name: "prod", Credential: "sk-test"})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	if account.ProxyMode != "off" || account.Proxy != nil {
		t.Fatalf("account must start off without proxy: %+v", account)
	}

	if _, err := providers.SetProxyMode(ctx, tenantID, providerID, account.ID, "auto"); !errors.Is(err, admin.ErrInvalidInput) {
		t.Fatalf("expected invalid proxy mode, got %v", err)
	}
	mode, err := providers.SetProxyMode(ctx, tenantID, providerID, account.ID, "preferred")
	if err != nil || mode != "preferred" {
		t.Fatalf("set proxy mode: %q err=%v", mode, err)
	}

	endpoint, err := proxies.Create(ctx, tenantID, admin.CreateProxyEndpointInput{
		Label: "SP", Address: "203.0.113.10", Port: 8080,
		CountryCode: "br", CityName: "São Paulo", Username: "proxy-user", Password: "proxy-pass",
	})
	if err != nil {
		t.Fatalf("create proxy: %v", err)
	}
	encoded, _ := json.Marshal(endpoint)
	if strings.Contains(string(encoded), "proxy-user") || strings.Contains(string(encoded), "proxy-pass") ||
		strings.Contains(string(encoded), "username") || strings.Contains(string(encoded), "password") {
		t.Fatalf("proxy credentials leaked: %s", encoded)
	}

	if err := proxies.Assign(ctx, tenantID, providerID, account.ID, endpoint.ID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	provider, err := providers.Get(ctx, tenantID, providerID)
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if len(provider.Accounts) != 1 || provider.Accounts[0].Proxy == nil {
		t.Fatalf("missing assigned proxy: %+v", provider.Accounts)
	}
	assigned := provider.Accounts[0].Proxy
	if assigned.Address != "203.0.113.10" || assigned.Port != 8080 || assigned.Health != "healthy" ||
		assigned.CityName != "São Paulo" || provider.Accounts[0].ProxyMode != "preferred" {
		t.Fatalf("unexpected proxy summary: %+v", assigned)
	}
	encoded, _ = json.Marshal(provider.Accounts[0])
	if strings.Contains(string(encoded), "proxy-user") || strings.Contains(string(encoded), "proxy-pass") {
		t.Fatalf("account leaked proxy credentials: %s", encoded)
	}

	if _, err := pool.Exec(ctx, `UPDATE proxy_endpoints SET quarantined_until = now() + interval '1 hour' WHERE id = $1::uuid`, endpoint.ID); err != nil {
		t.Fatalf("quarantine: %v", err)
	}
	endpoints, err := proxies.List(ctx, tenantID)
	if err != nil || len(endpoints) != 1 || endpoints[0].Health != "quarantined" {
		t.Fatalf("expected quarantined endpoint: %+v err=%v", endpoints, err)
	}
	if len(endpoints[0].Accounts) != 1 || endpoints[0].Accounts[0].Name != "prod" || endpoints[0].Accounts[0].ProxyMode != "preferred" {
		t.Fatalf("expected assigned account summary: %+v", endpoints[0].Accounts)
	}
	provider, _ = providers.Get(ctx, tenantID, providerID)
	if provider.Accounts[0].Proxy.Health != "quarantined" {
		t.Fatalf("expected quarantined account proxy, got %+v", provider.Accounts[0].Proxy)
	}

	if err := proxies.Unassign(ctx, tenantID, providerID, account.ID); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	provider, _ = providers.Get(ctx, tenantID, providerID)
	if provider.Accounts[0].Proxy != nil {
		t.Fatalf("expected no proxy after unassign, got %+v", provider.Accounts[0].Proxy)
	}
}

func TestProxyAssignmentLimitAndTenantIsolation(t *testing.T) {
	url := testDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	box, _ := secret.NewBox(make([]byte, 32))
	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
	const tenantID = "00000000-0000-0000-0000-000000000000"

	var providerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, 'openai', 'openai', 'native', true) RETURNING id::text
	`, tenantID).Scan(&providerID); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	accounts := make([]admin.Account, 0, 4)
	for index := 0; index < 4; index++ {
		account, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{
			Name: "account-" + string(rune('a'+index)), Credential: "sk-test",
		})
		if err != nil {
			t.Fatalf("add account %d: %v", index, err)
		}
		accounts = append(accounts, account)
	}
	endpoint, err := proxies.Create(ctx, tenantID, admin.CreateProxyEndpointInput{Address: "198.51.100.7", Port: 9000})
	if err != nil {
		t.Fatalf("create proxy: %v", err)
	}
	for index := 0; index < 3; index++ {
		if err := proxies.Assign(ctx, tenantID, providerID, accounts[index].ID, endpoint.ID); err != nil {
			t.Fatalf("assign %d: %v", index, err)
		}
	}
	if err := proxies.Assign(ctx, tenantID, providerID, accounts[3].ID, endpoint.ID); !errors.Is(err, admin.ErrProxyLimit) {
		t.Fatalf("expected proxy limit, got %v", err)
	}
	if err := proxies.Unassign(ctx, tenantID, providerID, accounts[0].ID); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	if err := proxies.Assign(ctx, tenantID, providerID, accounts[3].ID, endpoint.ID); err != nil {
		t.Fatalf("assign after freeing slot: %v", err)
	}

	var otherTenant string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tenants (id, name) VALUES (gen_random_uuid(), 'other') RETURNING id::text
	`).Scan(&otherTenant); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	var otherProvider string
	if err := pool.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, 'openai', 'openai', 'native', true) RETURNING id::text
	`, otherTenant).Scan(&otherProvider); err != nil {
		t.Fatalf("insert other provider: %v", err)
	}
	otherAccount, err := providers.AddAccount(ctx, otherTenant, otherProvider, admin.AccountInput{Name: "prod", Credential: "sk-other"})
	if err != nil {
		t.Fatalf("add other account: %v", err)
	}
	if err := proxies.Assign(ctx, otherTenant, otherProvider, otherAccount.ID, endpoint.ID); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected cross-tenant proxy to be rejected, got %v", err)
	}
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	return url
}
