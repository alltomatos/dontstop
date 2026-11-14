package mcp_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/mcp"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestServiceUsesPersistedToolOverridesAndAudit(t *testing.T) {
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
	store := admin.NewMCPService(pool)
	service := mcp.NewService(mcp.Dependencies{Store: store})
	principal := auth.Principal{TenantID: tenantID, KeyID: "key", Scopes: []string{
		"chat:write", "models:read", "embeddings:write", "web:read", "usage:read", "combos:read", "combos:write",
	}}
	tools, err := service.AllowedTools(ctx, principal)
	if err != nil {
		t.Fatalf("allowed tools: %v", err)
	}
	if len(tools) != 15 {
		t.Fatalf("expected 15 tools, got %d", len(tools))
	}
	if _, err := store.UpdateTool(ctx, tenantID, "test", "dontstop_chat", false); err != nil {
		t.Fatalf("disable tool: %v", err)
	}
	tools, err = service.AllowedTools(ctx, principal)
	if err != nil {
		t.Fatalf("allowed tools after override: %v", err)
	}
	if len(tools) != 14 {
		t.Fatalf("expected 14 tools, got %d", len(tools))
	}
	if err := store.RecordAudit(ctx, tenantID, "key", "mcp.tool.call", map[string]any{"tool": "dontstop_health", "status": "ok", "duration_ms": int64(4)}); err != nil {
		t.Fatalf("record audit: %v", err)
	}
	activity, err := store.Activity(ctx, tenantID, 24)
	if err != nil {
		t.Fatalf("activity: %v", err)
	}
	if activity.TotalCalls != 1 || activity.SuccessRate != 100 || len(activity.TopTools) != 1 || activity.TopTools[0].Tool != "dontstop_health" {
		t.Fatalf("unexpected activity: %+v", activity)
	}
}
