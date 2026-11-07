package config_test

import (
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/platform/config"
)

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Port != 8080 {
		t.Fatalf("expected port 8080, got %d", cfg.Port)
	}
}

func TestLoadReadsDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/dontstop?sslmode=disable")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/dontstop?sslmode=disable" {
		t.Fatalf("unexpected database URL: %q", cfg.DatabaseURL)
	}
}

func TestLoadReadsRedisURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Fatalf("unexpected Redis URL: %q", cfg.RedisURL)
	}
}

func TestLoadReadsMasterKey(t *testing.T) {
	t.Setenv("MASTER_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MasterKey != "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" {
		t.Fatalf("unexpected master key")
	}
}

func TestLoadReadsNATSURL(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.NATSURL != "nats://localhost:4222" {
		t.Fatalf("unexpected NATS URL: %q", cfg.NATSURL)
	}
}
func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-port")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected an error for an invalid port")
	}
}
