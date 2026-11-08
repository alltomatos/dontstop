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

func TestDLPServiceCreatesRulesAndListsEvents(t *testing.T) {
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
	service := admin.NewDLPService(pool)
	rule, err := service.CreateRule(ctx, tenantID, admin.DLPInput{
		Name: "CPF", Detector: "cpf", Action: "redact", Severity: "high",
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if rule.ID == "" || rule.Action != "redact" {
		t.Fatalf("unexpected rule: %+v", rule)
	}
	rules, err := service.Rules(ctx, tenantID)
	if err != nil || len(rules) != 1 {
		t.Fatalf("unexpected rules: %+v %v", rules, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO dlp_events (tenant_id, detector, action, severity, value_hash) VALUES ($1, 'cpf', 'redact', 'high', 'hash')`, tenantID); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	events, err := service.Events(ctx, tenantID, 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("unexpected events: %+v %v", events, err)
	}
}
