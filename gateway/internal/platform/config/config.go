package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultPort = 8080

// defaultCacheL1TTL keeps the in-process cache short enough that a stale entry
// can never outlive the invalidation window that matters for chat traffic.
const defaultCacheL1TTL = 10 * time.Second

func secretValue(name string) string {
	if path := os.Getenv(name + "_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
	return os.Getenv(name)
}

// Config contains process configuration loaded from environment variables.
type Config struct {
	Port                 int
	DatabaseURL          string
	RedisURL             string
	MasterKey            string
	NATSURL              string
	AgentRouterRelayURL  string
	WebshareSyncInterval time.Duration
	MaxChatBodyBytes     int64
	CacheL1TTL           time.Duration
	CacheL1MaxEntries    int
}

// Load reads and validates process configuration.
func Load() (Config, error) {
	cfg := Config{
		Port:                 defaultPort,
		DatabaseURL:          secretValue("DATABASE_URL"),
		RedisURL:             secretValue("REDIS_URL"),
		MasterKey:            secretValue("MASTER_KEY"),
		NATSURL:              secretValue("NATS_URL"),
		AgentRouterRelayURL:  strings.TrimSpace(secretValue("AGENTROUTER_RELAY_URL")),
		WebshareSyncInterval: time.Hour,
		CacheL1TTL:           defaultCacheL1TTL,
	}
	if raw := os.Getenv("CACHE_L1_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl < 0 {
			return Config{}, fmt.Errorf("invalid CACHE_L1_TTL %q: expected a non-negative duration", raw)
		}
		cfg.CacheL1TTL = ttl
	}
	if raw := os.Getenv("CACHE_L1_MAX_ENTRIES"); raw != "" {
		entries, err := strconv.Atoi(raw)
		if err != nil || entries <= 0 {
			return Config{}, fmt.Errorf("invalid CACHE_L1_MAX_ENTRIES %q: expected a positive integer", raw)
		}
		cfg.CacheL1MaxEntries = entries
	}
	if raw := os.Getenv("MAX_CHAT_BODY_BYTES"); raw != "" {
		bytes, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || bytes <= 0 {
			return Config{}, fmt.Errorf("invalid MAX_CHAT_BODY_BYTES %q: expected a positive integer", raw)
		}
		cfg.MaxChatBodyBytes = bytes
	}
	if raw := os.Getenv("WEBSHARE_SYNC_INTERVAL"); raw != "" {
		interval, err := time.ParseDuration(raw)
		if err != nil || interval <= 0 {
			return Config{}, fmt.Errorf("invalid WEBSHARE_SYNC_INTERVAL %q: expected a positive duration", raw)
		}
		cfg.WebshareSyncInterval = interval
	}
	rawPort := os.Getenv("PORT")
	if rawPort == "" {
		return cfg, nil
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT %q: expected an integer between 1 and 65535", rawPort)
	}
	cfg.Port = port
	return cfg, nil
}
