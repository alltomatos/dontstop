package cache_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/redis/go-redis/v9"
)

func redisCache(t *testing.T) (*cache.RedisExactCache, *redis.Client) {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is not set")
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis URL: %v", err)
	}
	client := redis.NewClient(options)
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return cache.NewRedisExactCache(client), client
}

func keyFor(t *testing.T, tenant, model string) string {
	t.Helper()
	key, err := cache.BuildKey(cache.KeyInput{TenantID: tenant, Model: model, Messages: []provider.ChatMessage{{Role: "user", Content: "Oi"}}})
	if err != nil {
		t.Fatalf("build key: %v", err)
	}
	return key
}

func TestRedisExactCacheRoundTripAndTenantIsolation(t *testing.T) {
	cacheClient, _ := redisCache(t)
	ctx := context.Background()
	key := keyFor(t, "tenant-a", "gpt-4.1")
	entry := cache.Entry{Response: provider.ChatResponse{ID: "chat-1", Content: "Olá"}}
	if err := cacheClient.Set(ctx, key, cache.Meta{TenantID: "tenant-a", Model: "gpt-4.1", Provider: "openai", PromptVersion: "v1"}, entry, time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, found, err := cacheClient.Get(ctx, key)
	if err != nil || !found || got.Response.Content != "Olá" {
		t.Fatalf("get: %+v found=%v err=%v", got, found, err)
	}
	if _, found, _ := cacheClient.Get(ctx, keyFor(t, "tenant-b", "gpt-4.1")); found {
		t.Fatal("tenant isolation failed")
	}
}

func TestRedisExactCacheInvalidation(t *testing.T) {
	cacheClient, _ := redisCache(t)
	ctx := context.Background()
	keys := map[string]cache.Meta{
		keyFor(t, "tenant-a", "gpt-4.1"):         {TenantID: "tenant-a", Model: "gpt-4.1", Provider: "openai", PromptVersion: "v1"},
		keyFor(t, "tenant-a", "claude-sonnet-4"): {TenantID: "tenant-a", Model: "claude-sonnet-4", Provider: "anthropic", PromptVersion: "v1"},
	}
	for key, meta := range keys {
		if err := cacheClient.Set(ctx, key, meta, cache.Entry{Response: provider.ChatResponse{Content: "x"}}, time.Minute); err != nil {
			t.Fatalf("set: %v", err)
		}
	}
	removed, err := cacheClient.Invalidate(ctx, "tenant-a", "provider", "openai")
	if err != nil || removed != 1 {
		t.Fatalf("invalidate provider: removed=%d err=%v", removed, err)
	}
	for key, meta := range keys {
		_, found, _ := cacheClient.Get(ctx, key)
		if meta.Provider == "openai" && found {
			t.Fatal("openai entry must be removed")
		}
		if meta.Provider == "anthropic" && !found {
			t.Fatal("anthropic entry must remain")
		}
	}
	if removed, err := cacheClient.Invalidate(ctx, "tenant-a", "tenant", ""); err != nil || removed != 1 {
		t.Fatalf("invalidate tenant: removed=%d err=%v", removed, err)
	}
}
