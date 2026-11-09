package api

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestLoginLimiterSharesStateThroughRedis(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse Redis URL: %v", err)
	}
	client := redis.NewClient(options)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping Redis: %v", err)
	}

	email := fmt.Sprintf("redis-limit-%d@example.com", time.Now().UnixNano())
	first := newLoginLimiter(client)
	second := newLoginLimiter(client)
	defer first.Reset(email)

	for attempt := 0; attempt < loginMaxAttempts; attempt++ {
		first.Fail(email)
	}
	if second.Allow(email) {
		t.Fatal("second replica must see the shared lockout")
	}
	second.Reset(email)
	if !first.Allow(email) {
		t.Fatal("reset must clear the shared lockout")
	}
}
