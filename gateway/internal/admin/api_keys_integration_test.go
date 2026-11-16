package admin_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestAPIKeyServiceLifecycle(t *testing.T) {
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
	service := admin.NewAPIKeyService(pool)
	rpm, tpm := 120, 5000
	monthly, daily, weekly := 50.0, 5.0, 20.0

	created, err := service.Create(ctx, tenantID, "actor", admin.CreateAPIKeyInput{
		Name: "console", Scopes: []string{"manage", "chat:write"},
		IPAllowlist: []string{"192.0.2.10", "2001:db8::/32"},
		RPMLimit: &rpm, TPMLimit: &tpm,
		USDMonthlyLimit: &monthly, USDDailyLimit: &daily, USDWeeklyLimit: &weekly,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Key == "" || created.Prefix == "" || created.Status != "active" {
		t.Fatalf("unexpected created key: %+v", created)
	}
	if created.ExpiresAt != nil || created.LastRotatedAt != nil {
		t.Fatalf("expected no expiration and no rotation: %+v", created)
	}
	if len(created.IPAllowlist) != 2 || created.RPMLimit == nil || *created.RPMLimit != rpm {
		t.Fatalf("unexpected allowlist/limits: %+v", created)
	}

	keys, err := service.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0].Status != "active" || keys[0].Name != "console" {
		t.Fatalf("unexpected keys: %+v", keys)
	}

	updated, err := service.Update(ctx, tenantID, "actor", created.ID, admin.UpdateAPIKeyInput{
		Name: "renamed", Scopes: []string{"chat:write"},
		IPAllowlist: []string{}, RPMLimit: nil, TPMLimit: &tpm,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "renamed" || len(updated.Scopes) != 1 || updated.RPMLimit != nil || updated.TPMLimit == nil {
		t.Fatalf("unexpected updated key: %+v", updated)
	}

	rotated, err := service.Rotate(ctx, tenantID, "actor", created.ID)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if rotated.Key == created.Key || rotated.Prefix == created.Prefix {
		t.Fatal("rotate did not replace the secret")
	}
	if rotated.LastRotatedAt == nil || rotated.Name != "renamed" || len(rotated.Scopes) != 1 {
		t.Fatalf("rotation did not preserve metadata: %+v", rotated)
	}

	revoked, err := service.Revoke(ctx, tenantID, "actor", created.ID)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if revoked.Status != "revoked" || revoked.RevokedAt == nil {
		t.Fatalf("expected revoked key: %+v", revoked)
	}
	if _, err := service.Rotate(ctx, tenantID, "actor", created.ID); !errors.Is(err, admin.ErrConflict) {
		t.Fatalf("expected conflict rotating revoked key, got %v", err)
	}

	expired, err := service.Create(ctx, tenantID, "actor", admin.CreateAPIKeyInput{
		Name: "expired", Scopes: []string{"models:read"},
	})
	if err != nil {
		t.Fatalf("create expired fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE api_keys SET expires_at = now() - interval '1 minute' WHERE id = $1::uuid`, expired.ID); err != nil {
		t.Fatalf("expire fixture: %v", err)
	}
	keys, err = service.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("list expired: %v", err)
	}
	foundExpired := false
	for _, key := range keys {
		if key.ID == expired.ID && key.Status == "expired" {
			foundExpired = true
		}
	}
	if !foundExpired {
		t.Fatalf("expected expired status: %+v", keys)
	}

	if _, err := service.Create(ctx, tenantID, "actor", admin.CreateAPIKeyInput{Name: "bad", Scopes: []string{"unknown:scope"}}); !errors.Is(err, admin.ErrInvalidInput) {
		t.Fatalf("expected invalid scope error, got %v", err)
	}
	if _, err := service.Create(ctx, tenantID, "actor", admin.CreateAPIKeyInput{Name: "bad", Scopes: []string{"chat:write"}, IPAllowlist: []string{"not-an-ip"}}); !errors.Is(err, admin.ErrInvalidInput) {
		t.Fatalf("expected invalid IP error, got %v", err)
	}

	var auditEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_outbox WHERE event_type IN ('api_key.create','api_key.update','api_key.rotate','api_key.revoke')`).Scan(&auditEvents); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if auditEvents < 4 {
		t.Fatalf("expected create/update/rotate/revoke audit events, got %d", auditEvents)
	}
	var leaked int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_outbox WHERE payload::text LIKE '%sk-ds-%'`).Scan(&leaked); err != nil {
		t.Fatalf("check secret leak: %v", err)
	}
	if leaked != 0 {
		t.Fatalf("audit payload contains an API key secret")
	}

	scopes := admin.Scopes()
	if len(scopes) == 0 {
		t.Fatal("expected a non-empty scope catalog")
	}
}
