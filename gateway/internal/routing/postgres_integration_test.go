package routing_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestPostgresRepositoryListsEligibleAccountsInPriorityOrder(t *testing.T) {
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
	_, err = pool.Exec(ctx, `
		INSERT INTO providers (tenant_id, name, kind, enabled) VALUES
			($1, 'openai', 'openai', true),
			($1, 'anthropic', 'anthropic', true),
			($1, 'disabled', 'openai', false)
	`, tenantID)
	if err != nil {
		t.Fatalf("insert providers: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, base_url, credential_cipher, priority, weight, status, cooldown_until) VALUES
			($1, (SELECT id FROM providers WHERE name='openai'), 'openai-primary', 'https://openai.example/v1', '\x01'::bytea, 20, 1, 'active', NULL),
			($1, (SELECT id FROM providers WHERE name='anthropic'), 'anthropic-primary', 'https://anthropic.example', '\x02'::bytea, 10, 1, 'active', NULL),
			($1, (SELECT id FROM providers WHERE name='openai'), 'openai-cooldown', 'https://openai.example/v1', '\x03'::bytea, 5, 1, 'active', now() + interval '5 minutes'),
			($1, (SELECT id FROM providers WHERE name='disabled'), 'disabled-account', 'https://disabled.example', '\x04'::bytea, 1, 1, 'active', NULL)
	`, tenantID)
	if err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
		SELECT $1::uuid, id, 'shared-model', 'Shared', true FROM providers WHERE name IN ('openai', 'anthropic')
		UNION ALL
		SELECT $1::uuid, id, 'shared-model', 'Disabled', true FROM providers WHERE name = 'disabled'
	`, tenantID)
	if err != nil {
		t.Fatalf("insert models: %v", err)
	}

	repository := routing.NewPostgresRepository(pool)
	candidates, err := repository.Candidates(ctx, tenantID, "shared-model")
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d: %+v", len(candidates), candidates)
	}
	if candidates[0].AccountID == "" || candidates[0].ProviderName != "anthropic" {
		t.Fatalf("expected anthropic first, got %+v", candidates[0])
	}
	if candidates[1].ProviderName != "openai" {
		t.Fatalf("expected openai second, got %+v", candidates[1])
	}
	if len(candidates[0].CredentialCipher) == 0 {
		t.Fatal("credential cipher was not loaded")
	}
	if candidates[0].BaseURL != "https://anthropic.example" || candidates[0].CredentialKeyVersion != 1 {
		t.Fatalf("unexpected endpoint metadata: %+v", candidates[0])
	}
}
