package config

import (
	"fmt"
	"os"
	"strconv"
)

const defaultPort = 8080

// Config contains process configuration loaded from environment variables.
type Config struct {
	Port        int
	DatabaseURL string
	RedisURL    string
	MasterKey   string
	NATSURL     string
}

// Load reads and validates process configuration.
func Load() (Config, error) {
	cfg := Config{
		Port:        defaultPort,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		MasterKey:   os.Getenv("MASTER_KEY"),
		NATSURL:     os.Getenv("NATS_URL"),
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
