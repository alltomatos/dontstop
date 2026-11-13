package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultPort = 8080

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
