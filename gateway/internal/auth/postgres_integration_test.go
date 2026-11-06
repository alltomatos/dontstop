package auth_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestPostgresRepositoryFindsOnlyActiveKeys(t *testing.T) {
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

	const key = "sk-ds-0123456789abcdef"
	hash, err := auth.HashKey(key)
	if err != nil {
		t.Fatalf("hash key: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO api_keys (tenant_id, name, key_prefix, key_hash, scopes)
		VALUES ('00000000-0000-0000-0000-000000000000', 'test', 'sk-ds-012345', $1, '["chat:write","models:read"]'::jsonb)
	`, hash)
	if err != nil {
		t.Fatalf("insert key: %v", err)
	}

	repository := auth.NewPostgresRepository(pool)
	record, err := repository.FindByPrefix(ctx, "sk-ds-012345")
	if err != nil {
		t.Fatalf("find key: %v", err)
	}
	if record.ID == "" || record.TenantID != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if len(record.Scopes) != 2 || record.Scopes[0] != "chat:write" {
		t.Fatalf("unexpected scopes: %+v", record.Scopes)
	}

	if _, err := repository.FindByPrefix(ctx, "sk-ds-unknown"); !errors.Is(err, auth.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	past := time.Now().Add(-time.Hour).UTC()
	if _, err := pool.Exec(ctx, "UPDATE api_keys SET revoked_at = $1", past); err != nil {
		t.Fatalf("revoke key: %v", err)
	}
	if _, err := repository.FindByPrefix(ctx, "sk-ds-012345"); !errors.Is(err, auth.ErrKeyNotFound) {
		t.Fatalf("revoked key must not be found, got %v", err)
	}
}
