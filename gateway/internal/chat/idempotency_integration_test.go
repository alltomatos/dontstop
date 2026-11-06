package chat_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/redis/go-redis/v9"
)

func TestRedisIdempotencyStoreRoundTripAndTenantScope(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis URL: %v", err)
	}
	client := redis.NewClient(options)
	defer func() { _ = client.Close() }()
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}

	store := chat.NewRedisIdempotencyStore(client, time.Minute)
	response := provider.ChatResponse{ID: "chat-1", Model: "gpt-4.1", Content: "Olá"}
	if err := store.Set(context.Background(), "tenant-a:key-1", response); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, found, err := store.Get(context.Background(), "tenant-a:key-1")
	if err != nil || !found || got.Content != "Olá" {
		t.Fatalf("unexpected get: %+v found=%v err=%v", got, found, err)
	}
	if _, found, _ := store.Get(context.Background(), "tenant-b:key-1"); found {
		t.Fatal("tenant B must not read tenant A idempotency response")
	}
}
