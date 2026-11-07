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

func TestMCPServicePersistsSettingsAndTools(t *testing.T) {
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
	service := admin.NewMCPService(pool)
	status, err := service.Status(ctx, tenantID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Enabled || status.Transport != "stdio" || len(status.Tools) != 6 {
		t.Fatalf("unexpected defaults: %+v", status)
	}
	enabled := true
	transport := "sse"
	status, err = service.UpdateSettings(ctx, tenantID, &enabled, &transport)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if !status.Enabled || status.Transport != "sse" {
		t.Fatalf("unexpected settings: %+v", status)
	}
	status, err = service.UpdateTool(ctx, tenantID, "dontstop_chat", false)
	if err != nil {
		t.Fatalf("update tool: %v", err)
	}
	for _, tool := range status.Tools {
		if tool.ID == "dontstop_chat" && tool.Enabled {
			t.Fatal("expected dontstop_chat disabled")
		}
	}
}
