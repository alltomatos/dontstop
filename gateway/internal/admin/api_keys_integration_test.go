package admin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestAPIKeyServiceCreatesListsAndRevokes(t *testing.T) {
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
	service := admin.NewAPIKeyService(pool)
	created, err := service.Create(ctx, tenantID, admin.CreateAPIKeyInput{
		Name: "console", Scopes: []string{"manage", "chat:write"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Key == "" || created.Prefix == "" || len(created.Scopes) != 2 {
		t.Fatalf("unexpected created key: %+v", created)
	}
	keys, err := service.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0].Name != "console" || keys[0].RevokedAt != nil {
		t.Fatalf("unexpected keys: %+v", keys)
	}
	revoked, err := service.Revoke(ctx, tenantID, created.ID)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}
}
