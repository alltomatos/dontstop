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
	"github.com/alltomatos/dontstop/gateway/internal/webshare"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

type fakeWebshare struct {
	proxies []webshare.Proxy
	err     error
}

func (f *fakeWebshare) ListDirect(context.Context, string, string) ([]webshare.Proxy, error) {
	return f.proxies, f.err
}

func (f *fakeWebshare) DiscoverPlanID(context.Context, string) (string, error) {
	return "plan-1", nil
}

func (f *fakeWebshare) Config(context.Context, string, string) (webshare.Config, error) {
	if f.err != nil {
		return webshare.Config{}, f.err
	}
	return webshare.Config{PlanID: "plan-1"}, nil
}

func (f *fakeWebshare) Status(context.Context, string, string) (webshare.Status, error) {
	if f.err != nil {
		return webshare.Status{}, f.err
	}
	return webshare.Status{State: "active", ProxyCount: len(f.proxies)}, nil
}

func TestWebshareCredentialSyncAndRotationPreserveAssignment(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
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
	const tenantID = "00000000-0000-0000-0000-000000000000"
	api := &fakeWebshare{proxies: []webshare.Proxy{
		{ID: "p-1", Address: "203.0.113.10", Port: 8080, Username: "u1", Password: "p1", Valid: true, CountryCode: "BR", CityName: "Sao Paulo"},
		{ID: "p-2", Address: "203.0.113.11", Port: 8081, Username: "u2", Password: "p2", Valid: true, CountryCode: "US", CityName: "New York"},
	}}
	service := admin.NewWebshareService(pool, box, api)

	status, err := service.Save(ctx, tenantID, "admin-1", admin.WebshareSaveInput{APIKey: "super-secret", PlanID: "plan-1"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !status.Configured || status.KeyVersion != 1 || status.PlanID != "plan-1" {
		t.Fatalf("unexpected status: %+v", status)
	}
	encoded, _ := json.Marshal(status)
	if strings.Contains(string(encoded), "super-secret") {
		t.Fatalf("status leaked the api key: %s", encoded)
	}
	var cipher []byte
	var rawKey string
	if err := pool.QueryRow(ctx, `SELECT api_key_cipher, encode(api_key_cipher, 'hex') FROM webshare_settings WHERE tenant_id = $1`, tenantID).Scan(&cipher, &rawKey); err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if strings.Contains(rawKey, "73757065722d736563726574") || len(cipher) == 0 {
		t.Fatalf("api key was not encrypted at rest")
	}
	var audits int
	var auditPayload string
	if err := pool.QueryRow(ctx, `SELECT count(*), coalesce(max(payload::text), '') FROM audit_outbox WHERE event_type = 'webshare.configure'`).Scan(&audits, &auditPayload); err != nil || audits != 1 {
		t.Fatalf("expected configuration audit, got %d err=%v", audits, err)
	}
	if strings.Contains(auditPayload, "super-secret") {
		t.Fatalf("audit leaked the api key: %s", auditPayload)
	}
	var otherTenant string
	if err := pool.QueryRow(ctx, `INSERT INTO tenants (id, name) VALUES (gen_random_uuid(), 'other') RETURNING id::text`).Scan(&otherTenant); err != nil {
		t.Fatalf("insert other tenant: %v", err)
	}
	otherStatus, err := service.Status(ctx, otherTenant)
	if err != nil || otherStatus.Configured {
		t.Fatalf("tenant leak: other tenant sees webshare settings: %+v err=%v", otherStatus, err)
	}
	if _, err := service.Sync(ctx, otherTenant, "admin-1"); !errors.Is(err, admin.ErrInvalidInput) {
		t.Fatalf("expected other tenant sync to be rejected, got %v", err)
	}

	result, err := service.Sync(ctx, tenantID, "admin-1")
	if err != nil || result.Imported != 2 || result.Status != "succeeded" {
		t.Fatalf("sync: %+v err=%v", result, err)
	}
	var endpointID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM proxy_endpoints WHERE tenant_id = $1 AND source = 'webshare' AND external_id = 'p-1'
	`, tenantID).Scan(&endpointID); err != nil {
		t.Fatalf("read imported proxy: %v", err)
	}

	providers := admin.NewProviderService(pool, box)
	proxies := admin.NewProxyService(pool, box)
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
	if err := proxies.Assign(ctx, tenantID, providerID, account.ID, endpointID); err != nil {
		t.Fatalf("assign: %v", err)
	}

	api.proxies[0].Address = "203.0.113.99"
	if _, err := service.Sync(ctx, tenantID, "admin-1"); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	provider, err := providers.Get(ctx, tenantID, providerID)
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if provider.Accounts[0].Proxy == nil || provider.Accounts[0].Proxy.Address != "203.0.113.99" {
		t.Fatalf("sync must update the assigned endpoint in place: %+v", provider.Accounts[0].Proxy)
	}

	api.proxies = api.proxies[:1]
	if _, err := service.Sync(ctx, tenantID, "admin-1"); err != nil {
		t.Fatalf("third sync: %v", err)
	}
	var valid bool
	if err := pool.QueryRow(ctx, `
		SELECT valid FROM proxy_endpoints WHERE tenant_id = $1 AND external_id = 'p-2'
	`, tenantID).Scan(&valid); err != nil || valid {
		t.Fatalf("missing proxy must become invalid: valid=%v err=%v", valid, err)
	}
	provider, _ = providers.Get(ctx, tenantID, providerID)
	if provider.Accounts[0].Proxy == nil {
		t.Fatal("sync must not drop an in-use assignment")
	}

	var storedPlan string
	if err := pool.QueryRow(ctx, `SELECT plan_id FROM webshare_settings WHERE tenant_id = $1`, tenantID).Scan(&storedPlan); err != nil || storedPlan != "plan-1" {
		t.Fatalf("unexpected plan: %q err=%v", storedPlan, err)
	}
}

func TestWebshareRejectsInvalidKeyWithoutPersisting(t *testing.T) {
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
	box, _ := secret.NewBox(make([]byte, 32))
	service := admin.NewWebshareService(pool, box, &fakeWebshare{err: webshare.ErrUnauthorized})
	_, err = service.Save(ctx, "00000000-0000-0000-0000-000000000000", "admin-1", admin.WebshareSaveInput{APIKey: "bad"})
	if !errors.Is(err, webshare.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM webshare_settings`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid key must not persist settings: count=%d err=%v", count, err)
	}
}
