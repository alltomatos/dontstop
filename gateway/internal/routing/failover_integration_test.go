package routing_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestRouteFailsOverAcrossAccountsOfSameProvider(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `
		INSERT INTO providers (tenant_id, name, kind, enabled) VALUES ($1, 'openai', 'openai', true)
	`, tenantID); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, base_url, credential_cipher, priority, weight, status)
		VALUES
			($1, (SELECT id FROM providers WHERE name='openai'), 'primary', 'https://primary.example', '\x01'::bytea, 10, 1, 'active'),
			($1, (SELECT id FROM providers WHERE name='openai'), 'secondary', 'https://secondary.example', '\x02'::bytea, 20, 1, 'active')
	`, tenantID); err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
		SELECT $1::uuid, id, 'shared-model', 'Shared', true FROM providers WHERE name='openai'
	`, tenantID); err != nil {
		t.Fatalf("insert model: %v", err)
	}

	repository := routing.NewPostgresRepository(pool)
	candidates, err := repository.Candidates(ctx, tenantID, "shared-model")
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	breaker := newRecordingBreaker()
	for _, candidate := range candidates {
		breaker.allowed[candidate.AccountID] = true
	}
	router := routing.NewRouter(repository, breaker)
	var attempts []string
	_, err = router.Route(ctx, tenantID, "shared-model", func(_ context.Context, account routing.Candidate) (routing.AttemptResult, error) {
		attempts = append(attempts, account.BaseURL)
		if account.BaseURL == "https://primary.example" {
			return routing.AttemptResult{Failure: routing.FailureRateLimit}, errors.New("429")
		}
		return routing.AttemptResult{}, nil
	})
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if len(attempts) != 2 || attempts[0] != "https://primary.example" || attempts[1] != "https://secondary.example" {
		t.Fatalf("unexpected attempts: %v", attempts)
	}
	for _, candidate := range candidates {
		if candidate.BaseURL == "https://primary.example" && breaker.failures[candidate.AccountID] != routing.FailureRateLimit {
			t.Fatalf("expected reason-specific cooldown, got %+v", breaker.failures)
		}
	}
}
