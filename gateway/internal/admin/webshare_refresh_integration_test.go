package admin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/internal/webshare"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestRefreshPoolDebouncesAndIgnoresUnconfigured(t *testing.T) {
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
	const tenantID = "00000000-0000-0000-0000-000000000000"
	api := &fakeWebshare{proxies: []webshare.Proxy{
		{ID: "p-1", Address: "203.0.113.60", Port: 8080, Username: "u1", Password: "p1", Valid: true},
	}}
	service := admin.NewWebshareService(pool, box, api, nil)

	if err := service.RefreshPool(ctx, "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("refresh without configuration must be a no-op: %v", err)
	}
	if api.listCalls != 0 {
		t.Fatalf("unconfigured refresh must not list proxies, got %d calls", api.listCalls)
	}

	if _, err := service.Save(ctx, tenantID, "admin-1", admin.WebshareSaveInput{APIKey: "super-secret", PlanID: "plan-1"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	api.listCalls = 0

	if err := service.RefreshPool(ctx, tenantID); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := service.RefreshPool(ctx, tenantID); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if api.listCalls != 1 {
		t.Fatalf("expected a debounced single refresh, got %d calls", api.listCalls)
	}
}
