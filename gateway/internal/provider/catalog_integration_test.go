package provider_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestCatalogListsActiveModelsFromConnectedProviders(t *testing.T) {
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
		INSERT INTO providers (tenant_id, name, kind, enabled, origin) VALUES
			($1, 'openai', 'openai', true, '9router'),
			($1, 'anthropic', 'anthropic', true, 'omniroute'),
			($1, 'disabled-provider', 'openai', false, 'allcode')
	`, tenantID)
	if err != nil {
		t.Fatalf("insert providers: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, credential_cipher, status, cooldown_until)
		SELECT $1, id, 'primary', '\x00'::bytea, 'active', NULL FROM providers WHERE name = 'openai'
	`, tenantID)
	if err != nil {
		t.Fatalf("insert OpenAI account: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, credential_cipher, status, cooldown_until)
		SELECT $1, id, 'primary', '\x00'::bytea, 'cooldown', now() + interval '5 minutes' FROM providers WHERE name = 'anthropic'
	`, tenantID)
	if err != nil {
		t.Fatalf("insert Anthropic account: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
		SELECT $1::uuid, id, 'gpt-4.1', 'GPT-4.1', true FROM providers WHERE name = 'openai'
		UNION ALL
		SELECT $1::uuid, id, 'gpt-4.1-mini', 'GPT-4.1 mini', true FROM providers WHERE name = 'openai'
		UNION ALL
		SELECT $1::uuid, id, 'gpt-3.5-turbo', 'GPT-3.5', false FROM providers WHERE name = 'openai'
		UNION ALL
		SELECT $1::uuid, id, 'claude-sonnet-4', 'Claude Sonnet 4', true FROM providers WHERE name = 'anthropic'
		UNION ALL
		SELECT $1::uuid, id, 'disabled-model', 'Disabled', true FROM providers WHERE name = 'disabled-provider'
	`, tenantID)
	if err != nil {
		t.Fatalf("insert models: %v", err)
	}

	catalog := provider.NewCatalog(pool)
	models, err := catalog.ListModels(ctx, tenantID)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d: %+v", len(models), models)
	}

	byID := make(map[string]provider.Model, len(models))
	for _, model := range models {
		byID[model.ID] = model
	}
	if byID["gpt-4.1"].Status != "active" || byID["gpt-4.1"].OwnedBy != "openai" || byID["gpt-4.1"].Origin != "9router" {
		t.Fatalf("unexpected gpt-4.1: %+v", byID["gpt-4.1"])
	}
	if byID["claude-sonnet-4"].Origin != "omniroute" {
		t.Fatalf("unexpected claude origin: %+v", byID["claude-sonnet-4"])
	}
	if byID["claude-sonnet-4"].Status != "cooldown" {
		t.Fatalf("expected cooldown model, got %+v", byID["claude-sonnet-4"])
	}
	if _, found := byID["gpt-3.5-turbo"]; found {
		t.Fatal("inactive model must not be listed")
	}
	if _, found := byID["disabled-model"]; found {
		t.Fatal("model from disabled provider must not be listed")
	}
}
