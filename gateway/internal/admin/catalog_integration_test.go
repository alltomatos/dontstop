package admin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestProviderCatalogSeedsAndConnects(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
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
	service := admin.NewProviderService(pool, box)
	// Keep the test hermetic: provider creation now imports advertised models.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"remote-alpha","name":"Remote Alpha"}]}`))
	}))
	defer upstream.Close()

	entries, err := service.ListCatalog(ctx)
	if err != nil {
		t.Fatalf("list catalog: %v", err)
	}
	if len(entries) != 16 {
		t.Fatalf("expected 16 catalog entries, got %d", len(entries))
	}
	bySlug := make(map[string]admin.CatalogEntry, len(entries))
	for _, entry := range entries {
		bySlug[entry.Slug] = entry
	}
	goEntry, ok := bySlug["opencode-go"]
	if !ok || goEntry.DefaultBaseURL != "https://opencode.ai/inference/go/openai/v1" || goEntry.ModelsPath != "/models" {
		t.Fatalf("unexpected opencode-go catalog entry: %+v", goEntry)
	}
	if entry, ok := bySlug["agentrouter"]; !ok || entry.Protocol != "agentrouter" || entry.ModelsPath != "" || entry.AuthHeader != "x-api-key" || len(entry.Models) < 5 {
		t.Fatalf("unexpected agentrouter catalog entry: %+v", entry)
	}
	if entry, ok := bySlug["anthropic"]; !ok || entry.Protocol != "anthropic" || entry.ModelsPath != "/v1/models" {
		t.Fatalf("unexpected anthropic catalog entry: %+v", entry)
	}
	if entry, ok := bySlug["perplexity"]; !ok || entry.ModelsPath != "" {
		t.Fatalf("perplexity must not declare a models endpoint: %+v", entry)
	}
	if len(bySlug["openai"].Models) == 0 {
		t.Fatal("openai catalog entry has no models")
	}

	const tenantID = "00000000-0000-0000-0000-000000000000"
	agent, err := service.CreateFromCatalog(ctx, tenantID, admin.CreateFromCatalogInput{Slug: "agentrouter", AccountName: "agent-main", Credential: "agent-key"})
	if err != nil {
		t.Fatalf("create agentrouter from catalog: %v", err)
	}
	if agent.Kind != "agentrouter" || agent.Name != "AgentRouter" || agent.ModelsCount < 5 || agent.Accounts[0].Name != "agent-main" || agent.Accounts[0].BaseURL != "https://agentrouter.org" {
		t.Fatalf("unexpected agentrouter provider: %+v", agent)
	}
	agentModels, err := service.ConfiguredModels(ctx, tenantID, agent.ID)
	if err != nil || len(agentModels) < 5 {
		t.Fatalf("agentrouter models: %+v err=%v", agentModels, err)
	}
	caps, _ := agentModels[0].Capabilities.(map[string]any)
	if caps["text_input"] != true || caps["tools"] != true || caps["stream"] != true || caps["metadata_source"] != "catalog:provider_catalog_models" {
		t.Fatalf("unexpected agentrouter capabilities: %+v", caps)
	}
	provider, err := service.CreateFromCatalog(ctx, tenantID, admin.CreateFromCatalogInput{
		Slug: "openrouter", AccountName: "openrouter-prod", Credential: "sk-or-test", BaseURL: upstream.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("create from catalog: %v", err)
	}
	if provider.Kind != "openai" || provider.CatalogSlug != "openrouter" || provider.Origin != "native" {
		t.Fatalf("unexpected created provider: %+v", provider)
	}
	if provider.AccountsCount != 1 || provider.ModelsCount < 2 {
		t.Fatalf("expected one account and imported models, got %d/%d", provider.AccountsCount, provider.ModelsCount)
	}
	if provider.Name != "OpenRouter" || provider.Accounts[0].Name != "openrouter-prod" || provider.Accounts[0].BaseURL != upstream.URL+"/v1" {
		t.Fatalf("unexpected account base_url: %q", provider.Accounts[0].BaseURL)
	}
	configured, err := service.ConfiguredModels(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("configured models: %v", err)
	}
	found := false
	for _, model := range configured {
		if model.ModelID == "remote-alpha" && model.Active {
			found = true
		}
	}
	if !found {
		t.Fatalf("advertised model was not imported: %+v", configured)
	}

	models, err := service.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("unexpected providers: %+v", models)
	}
	foundOpenRouter := false
	for _, item := range models {
		if item.CatalogSlug == "openrouter" {
			foundOpenRouter = true
		}
	}
	if !foundOpenRouter {
		t.Fatalf("openrouter must be configured: %+v", models)
	}

	if _, err := service.CreateFromCatalog(ctx, tenantID, admin.CreateFromCatalogInput{
		Slug: "openrouter", Credential: "sk-or-test",
	}); !errors.Is(err, admin.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate provider, got %v", err)
	}
	if _, err := service.CreateFromCatalog(ctx, tenantID, admin.CreateFromCatalogInput{
		Slug: "does-not-exist", Credential: "sk-test",
	}); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown slug, got %v", err)
	}
	if _, err := service.CreateFromCatalog(ctx, tenantID, admin.CreateFromCatalogInput{
		Slug: "openrouter", Credential: "sk-or-test", BaseURL: "file:///etc/passwd",
	}); !errors.Is(err, admin.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for non-HTTP base URL, got %v", err)
	}

	var slug string
	var kind string
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(p.catalog_slug, ''), p.kind
		FROM providers p WHERE p.tenant_id = $1 AND p.id = $2::uuid
	`, tenantID, provider.ID).Scan(&slug, &kind); err != nil {
		t.Fatalf("query created provider: %v", err)
	}
	if slug != "openrouter" || kind != "openai" {
		t.Fatalf("unexpected provider row: slug=%q kind=%q", slug, kind)
	}
}
