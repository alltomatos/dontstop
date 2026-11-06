package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestModelsEndpointEnforcesTenantAndScopeWithRealDatabase(t *testing.T) {
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

	const tenantA = "00000000-0000-0000-0000-000000000000"
	const tenantB = "11111111-1111-1111-1111-111111111111"
	if _, err := pool.Exec(ctx, "INSERT INTO tenants (id, name) VALUES ($1, 'Tenant B')", tenantB); err != nil {
		t.Fatalf("insert tenant B: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO providers (tenant_id, name, kind, enabled) VALUES ($1, 'openai', 'openai', true)
	`, tenantA); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, credential_cipher, status)
		SELECT $1, id, 'primary', '\x00'::bytea, 'active' FROM providers WHERE tenant_id = $1 AND name = 'openai'
	`, tenantA); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
		SELECT $1, id, 'gpt-4.1', 'GPT-4.1', true FROM providers WHERE tenant_id = $1 AND name = 'openai'
	`, tenantA); err != nil {
		t.Fatalf("insert model: %v", err)
	}

	keyA := "sk-ds-aaaa000000000001"
	keyB := "sk-ds-bbbb000000000001"
	keyNoScope := "sk-ds-cccc000000000001"
	insertKey := func(tenantID, key, scopes string) {
		t.Helper()
		hash, err := auth.HashKey(key)
		if err != nil {
			t.Fatalf("hash key: %v", err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO api_keys (tenant_id, name, key_prefix, key_hash, scopes)
			VALUES ($1, 'test', $2, $3, $4::jsonb)
		`, tenantID, key[:12], hash, scopes)
		if err != nil {
			t.Fatalf("insert key: %v", err)
		}
	}
	insertKey(tenantA, keyA, `["models:read"]`)
	insertKey(tenantB, keyB, `["models:read"]`)
	insertKey(tenantA, keyNoScope, `["chat:write"]`)

	verifier := auth.NewVerifier(auth.NewPostgresRepository(pool))
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(time.Second),
		Verifier:  verifier,
		Models:    provider.NewCatalog(pool),
	})

	request := func(header string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	if rec := request(""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing key: expected 401, got %d", rec.Code)
	}
	if rec := request("Bearer ' OR 1=1--xxxxxxxx"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("injection-like key: expected 401, got %d", rec.Code)
	}
	if rec := request("Bearer " + keyNoScope); rec.Code != http.StatusForbidden {
		t.Fatalf("missing scope: expected 403, got %d", rec.Code)
	}

	rec := request("Bearer " + keyA)
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant A: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode tenant A: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "gpt-4.1" {
		t.Fatalf("unexpected tenant A models: %+v", body.Data)
	}

	rec = request("Bearer " + keyB)
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant B: expected 200, got %d", rec.Code)
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode tenant B: %v", err)
	}
	if len(body.Data) != 0 {
		t.Fatalf("tenant B must not see tenant A models: %+v", body.Data)
	}
}
