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

func TestProviderAccountSummariesAndIsolatedModelTests(t *testing.T) {
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
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"remote-alpha","name":"Remote Alpha"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_1","object":"chat.completion","created":1762387200,"model":"remote-alpha",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer okServer.Close()
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"remote-alpha"}]}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
	}))
	defer failServer.Close()

	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	service := admin.NewProviderService(pool, box)
	const tenantID = "00000000-0000-0000-0000-000000000000"
	provider, err := service.Create(ctx, tenantID, admin.CreateProviderInput{
		Name: "Summary Test", Kind: "openai", Origin: "native", ModelID: "remote-alpha", Credential: "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	first, err := service.AddAccount(ctx, tenantID, provider.ID, admin.AccountInput{
		Name: "account-a", Credential: "sk-a", BaseURL: okServer.URL,
	})
	if err != nil {
		t.Fatalf("add account A: %v", err)
	}
	second, err := service.AddAccount(ctx, tenantID, provider.ID, admin.AccountInput{
		Name: "account-b", Credential: "sk-b", BaseURL: failServer.URL,
	})
	if err != nil {
		t.Fatalf("add account B: %v", err)
	}
	if err := service.RenameAccount(ctx, tenantID, provider.ID, first.ID, "account-a-renamed"); err != nil {
		t.Fatalf("rename account: %v", err)
	}
	renamed, err := service.Get(ctx, tenantID, provider.ID)
	if err != nil || renamed.Name != "Summary Test" {
		t.Fatalf("provider name must stay immutable: %+v err=%v", renamed, err)
	}
	if err := service.RenameAccount(ctx, tenantID, provider.ID, first.ID, "account-b"); !errors.Is(err, admin.ErrConflict) {
		t.Fatalf("expected duplicate account name conflict, got %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE provider_models SET capabilities = '{"tools": true, "reasoning": true, "text_input": true}'::jsonb
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND model_id = 'remote-alpha'
	`, tenantID, provider.ID); err != nil {
		t.Fatalf("set model capabilities: %v", err)
	}
	inherited, err := service.AddAccount(ctx, tenantID, provider.ID, admin.AccountInput{Name: "account-c", Credential: "sk-c"})
	if err != nil || inherited.BaseURL != okServer.URL {
		t.Fatalf("account without base URL must inherit the provider default: %+v err=%v", inherited, err)
	}

	summary, err := service.AccountSummary(ctx, tenantID, provider.ID, first.ID)
	if err != nil || summary.Account.Name != "account-a-renamed" || summary.Provider.ModelsUsable != 1 {
		t.Fatalf("unexpected account summary: %+v err=%v", summary, err)
	}

	result, err := service.TestModel(ctx, tenantID, provider.ID, "remote-alpha", first.ID)
	if err != nil || !result.OK || len(result.Accounts) != 1 || result.Accounts[0].AccountName != "account-a-renamed" {
		t.Fatalf("unexpected account A test: %+v err=%v", result, err)
	}
	broken, err := service.TestModel(ctx, tenantID, provider.ID, "remote-alpha", second.ID)
	if err != nil || broken.OK || len(broken.Accounts) != 1 || broken.Accounts[0].Error == "" {
		t.Fatalf("unexpected account B test: %+v err=%v", broken, err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE provider_accounts SET status = 'cooldown', cooldown_until = now() + interval '5 minutes'
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, second.ID); err != nil {
		t.Fatalf("cooldown account B: %v", err)
	}

	detail, err := service.Get(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if detail.UsableAccounts != 3 || detail.ModelsUsable != 1 {
		t.Fatalf("unexpected provider aggregate: %+v", detail)
	}
	summaries := map[string]admin.Account{}
	for _, account := range detail.Accounts {
		summaries[account.Name] = account
	}
	if summaries["account-a-renamed"].Health != "active" || !summaries["account-a-renamed"].Active ||
		summaries["account-a-renamed"].ModelsAvailable != 1 || summaries["account-a-renamed"].ModelsUsable != 1 ||
		len(summaries["account-a-renamed"].Capabilities) == 0 {
		t.Fatalf("unexpected active summary: %+v", summaries["account-a-renamed"])
	}
	if summaries["account-b"].Health != "cooldown" || summaries["account-b"].Active ||
		summaries["account-b"].ModelsAvailable != 1 || summaries["account-b"].ModelsUsable != 0 {
		t.Fatalf("unexpected cooldown summary: %+v", summaries["account-b"])
	}

	other, err := service.Create(ctx, tenantID, admin.CreateProviderInput{
		Name: "Other Provider", Kind: "openai", Origin: "native", ModelID: "remote-alpha", Credential: "sk-other",
	})
	if err != nil {
		t.Fatalf("create other provider: %v", err)
	}
	otherAccount, err := service.AddAccount(ctx, tenantID, other.ID, admin.AccountInput{
		Name: "other-account", Credential: "sk-other", BaseURL: okServer.URL,
	})
	if err != nil {
		t.Fatalf("add other account: %v", err)
	}
	if _, err := service.AccountSummary(ctx, tenantID, provider.ID, otherAccount.ID); err != admin.ErrNotFound {
		t.Fatalf("expected cross-provider summary to be rejected, got %v", err)
	}
	if _, err := service.TestModel(ctx, tenantID, provider.ID, "remote-alpha", otherAccount.ID); err != admin.ErrNotFound {
		t.Fatalf("expected cross-provider account to be rejected, got %v", err)
	}
	if _, err := service.TestModel(ctx, tenantID, provider.ID, "remote-alpha", "00000000-0000-0000-0000-000000000000"); err != admin.ErrNotFound {
		t.Fatalf("expected invalid account to be rejected, got %v", err)
	}
}
