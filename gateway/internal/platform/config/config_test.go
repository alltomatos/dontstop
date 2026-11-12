package config_test

import (
	"os"
	"testing"
	"time"

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

func TestLoadReadsWebshareInterval(t *testing.T) {
	t.Setenv("WEBSHARE_SYNC_INTERVAL", "15m")
	cfg, err := config.Load()
	if err != nil || cfg.WebshareSyncInterval != 15*time.Minute {
		t.Fatalf("unexpected interval: %s err=%v", cfg.WebshareSyncInterval, err)
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
func TestLoadReadsDockerSecretFiles(t *testing.T) {
	dir := t.TempDir()
	databaseFile := dir + "/database_url"
	redisFile := dir + "/redis_url"
	masterFile := dir + "/master_key"
	natsFile := dir + "/nats_url"
	for path, value := range map[string]string{
		databaseFile: "postgres://from-file/db",
		redisFile:    "redis://from-file:6379/0",
		masterFile:   "file-master-key",
		natsFile:     "nats://from-file:4222",
	} {
		if err := os.WriteFile(path, []byte(value+"\n"), 0o600); err != nil {
			t.Fatalf("write secret: %v", err)
		}
	}
	t.Setenv("DATABASE_URL_FILE", databaseFile)
	t.Setenv("REDIS_URL_FILE", redisFile)
	t.Setenv("MASTER_KEY_FILE", masterFile)
	t.Setenv("NATS_URL_FILE", natsFile)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DatabaseURL != "postgres://from-file/db" || cfg.RedisURL != "redis://from-file:6379/0" ||
		cfg.MasterKey != "file-master-key" || cfg.NATSURL != "nats://from-file:4222" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-port")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected an error for an invalid port")
	}
}
