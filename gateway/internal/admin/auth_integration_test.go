package admin_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestAuthServiceSetupLoginAndSession(t *testing.T) {
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

	service := admin.NewAuthService(pool)
	needsSetup, err := service.NeedsSetup(ctx)
	if err != nil || !needsSetup {
		t.Fatalf("expected setup to be required: %v %v", needsSetup, err)
	}

	session, err := service.Setup(ctx, admin.SetupInput{
		Name: "Owner", Email: "owner@example.com", Password: "change-me-123",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if session.Token == "" || session.User.Email != "owner@example.com" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if _, err := service.Setup(ctx, admin.SetupInput{
		Name: "Other", Email: "other@example.com", Password: "change-me-123",
	}); !errors.Is(err, admin.ErrSetupComplete) {
		t.Fatalf("expected ErrSetupComplete, got %v", err)
	}

	if _, err := service.Login(ctx, admin.LoginInput{Email: "owner@example.com", Password: "wrong"}); !errors.Is(err, admin.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	login, err := service.Login(ctx, admin.LoginInput{Email: "owner@example.com", Password: "change-me-123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	principal, err := service.Authenticate(ctx, login.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.TenantID != "00000000-0000-0000-0000-000000000000" || !principal.HasScope("manage") {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if err := service.Logout(ctx, login.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Authenticate(ctx, login.Token); !errors.Is(err, admin.ErrUnauthorized) {
		t.Fatalf("expected unauthorized after logout, got %v", err)
	}
}

func TestAuthServiceSetupIsAtomic(t *testing.T) {
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

	service := admin.NewAuthService(pool)
	emails := []string{"race-a@example.com", "race-b@example.com"}
	results := make(chan error, len(emails))
	var wait sync.WaitGroup
	for _, email := range emails {
		wait.Add(1)
		go func(email string) {
			defer wait.Done()
			_, err := service.Setup(ctx, admin.SetupInput{Name: "Owner", Email: email, Password: "change-me-123"})
			results <- err
		}(email)
	}
	wait.Wait()
	close(results)

	success, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, admin.ErrSetupComplete):
			conflicts++
		default:
			t.Fatalf("unexpected setup error: %v", err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got success=%d conflicts=%d", success, conflicts)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM admin_users").Scan(&count); err != nil {
		t.Fatalf("count admin users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one owner, got %d", count)
	}
}
