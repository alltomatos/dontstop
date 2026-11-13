package admin_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestComboLifecycleAndResolution(t *testing.T) {
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

	const tenantID = "00000000-0000-0000-0000-000000000000"
	var providerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, 'openai', 'openai', 'native', true)
		RETURNING id::text
	`, tenantID).Scan(&providerID); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	for _, model := range []string{"gpt-5-mini", "gpt-5"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
			VALUES ($1, $2::uuid, $3, $3, true)
		`, tenantID, providerID, model); err != nil {
			t.Fatalf("insert model: %v", err)
		}
	}
	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	providers := admin.NewProviderService(pool, box)
	if _, err := providers.AddAccount(ctx, tenantID, providerID, admin.AccountInput{
		Name: "primary", Credential: "sk-test", BaseURL: "https://api.openai.com",
	}); err != nil {
		t.Fatalf("add account: %v", err)
	}

	service := admin.NewComboService(pool)
	combo, err := service.Create(ctx, tenantID, admin.ComboInput{
		Slug: "fast-coding", Name: "Fast coding", Strategy: "priority_round_robin",
		FailoverEnabled: true, MaxAttempts: 5,
		Members: []admin.ComboMemberInput{
			{ProviderID: providerID, ModelID: "gpt-5-mini", Priority: 0, Position: 0, Weight: 60, Enabled: true},
			{ProviderID: providerID, ModelID: "gpt-5", Priority: 1, Position: 1, Weight: 100, Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("create combo: %v", err)
	}
	if combo.Status != "draft" || combo.MemberCount != 2 || combo.UsableMembers != 2 {
		t.Fatalf("unexpected combo: %+v", combo)
	}
	if _, err := service.SetStatus(ctx, tenantID, combo.ID, "active"); err != nil {
		t.Fatalf("activate combo: %v", err)
	}
	targets, revision, err := service.ResolveTargets(ctx, tenantID, "fast-coding")
	if err != nil {
		t.Fatalf("resolve combo: %v", err)
	}
	if revision < 2 || len(targets) != 2 || targets[0].ModelID != "gpt-5-mini" {
		t.Fatalf("unexpected targets: rev=%d %+v", revision, targets)
	}
	updated, err := service.Update(ctx, tenantID, combo.ID, combo.Revision+1, admin.ComboInput{
		Slug: "fast-coding", Name: "Fast coding v2", Strategy: "priority",
		FailoverEnabled: true, MaxAttempts: 4,
		Members: []admin.ComboMemberInput{
			{ProviderID: providerID, ModelID: "gpt-5", Priority: 0, Position: 0, Weight: 100, Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("update combo: %v", err)
	}
	if updated.Name != "Fast coding v2" || updated.MemberCount != 1 {
		t.Fatalf("unexpected updated combo: %+v", updated)
	}
	if _, err := service.Update(ctx, tenantID, combo.ID, combo.Revision, admin.ComboInput{
		Slug: "fast-coding", Name: "stale", Strategy: "priority",
		Members: []admin.ComboMemberInput{{ProviderID: providerID, ModelID: "gpt-5", Priority: 0, Position: 0, Weight: 1, Enabled: true}},
	}); !errors.Is(err, admin.ErrConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	if err := service.Delete(ctx, tenantID, combo.ID); err != nil {
		t.Fatalf("delete combo: %v", err)
	}
	if _, err := service.Get(ctx, tenantID, combo.ID); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("expected combo not found, got %v", err)
	}
}
