package admin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestProviderServiceListsAndAddsAccounts(t *testing.T) {
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
	if err := pool.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, 'openai', 'openai', 'native', true)
		RETURNING id::text
	`, tenantID).Scan(&providerID); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
		VALUES ($1, $2::uuid, 'gpt-4.1', 'GPT-4.1', true)
	`, tenantID, providerID); err != nil {
		t.Fatalf("insert model: %v", err)
	}

	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	service := admin.NewProviderService(pool, box)
	providers, err := service.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(providers) != 1 || providers[0].Origin != "native" || providers[0].ModelsCount != 1 {
		t.Fatalf("unexpected providers: %+v", providers)
	}

	account, err := service.AddAccount(ctx, tenantID, providerID, admin.AccountInput{
		Name: "prod", Credential: "sk-test", BaseURL: "https://api.openai.com",
	})
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	if account.ID == "" || account.Name != "prod" || account.Status != "active" {
		t.Fatalf("unexpected account: %+v", account)
	}
	provider, err := service.Get(ctx, tenantID, providerID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(provider.Accounts) != 1 || provider.Accounts[0].Name != "prod" {
		t.Fatalf("unexpected provider detail: %+v", provider)
	}
}

func TestProviderOriginRejectsReferenceGateways(t *testing.T) {
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
	for _, origin := range []string{"9router", "omniroute", "allcode"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO providers (tenant_id, name, kind, origin, enabled)
			VALUES ('00000000-0000-0000-0000-000000000000', $1, 'openai', $2, true)
		`, "invalid-"+origin, origin); err == nil {
			t.Fatalf("origin %q must be rejected", origin)
		}
	}
}
