package redisclient_test

import (
	"context"
	"os"
	"testing"

	redisclient "github.com/alltomatos/dontstop/gateway/internal/platform/redis"
)

func TestOpenPingsRedis(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}
	client, err := redisclient.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}
	defer func() { _ = client.Close() }()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping Redis: %v", err)
	}
}
