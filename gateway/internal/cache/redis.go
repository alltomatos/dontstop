package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/redis/go-redis/v9"
)

// Entry is a cached response or negative cache marker.
type Entry struct {
	Response  provider.ChatResponse `json:"response"`
	ErrorCode string                `json:"error_code,omitempty"`
}

// Meta tags a cache entry for invalidation.
type Meta struct {
	TenantID      string
	Model         string
	Provider      string
	PromptVersion string
}

// ExactCache stores and invalidates exact responses.
type ExactCache interface {
	Get(ctx context.Context, key string) (Entry, bool, error)
	Set(ctx context.Context, key string, meta Meta, entry Entry, ttl time.Duration) error
	Invalidate(ctx context.Context, tenantID, kind, value string) (int64, error)
}

// RedisExactCache is a Redis-backed exact cache.
type RedisExactCache struct {
	client redis.UniversalClient
}

// NewRedisExactCache creates a Redis exact cache.
func NewRedisExactCache(client redis.UniversalClient) *RedisExactCache {
	return &RedisExactCache{client: client}
}

// Get returns a cached entry.
func (c *RedisExactCache) Get(ctx context.Context, key string) (Entry, bool, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("read exact cache: %w", err)
	}
	var entry Entry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return Entry{}, false, fmt.Errorf("decode exact cache: %w", err)
	}
	return entry, true, nil
}

// Set stores an entry and its invalidation tags.
func (c *RedisExactCache) Set(ctx context.Context, key string, meta Meta, entry Entry, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Hour
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode exact cache: %w", err)
	}
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, key, raw, ttl)
	indexes := []string{
		c.indexKey(meta.TenantID, "all", ""),
		c.indexKey(meta.TenantID, "model", meta.Model),
		c.indexKey(meta.TenantID, "provider", meta.Provider),
		c.indexKey(meta.TenantID, "prompt", meta.PromptVersion),
	}
	for _, index := range indexes {
		if index == "" {
			continue
		}
		pipe.SAdd(ctx, index, key)
		pipe.Expire(ctx, index, ttl+time.Minute)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("store exact cache: %w", err)
	}
	return nil
}

// Invalidate removes entries by kind: model, provider, prompt or tenant.
func (c *RedisExactCache) Invalidate(ctx context.Context, tenantID, kind, value string) (int64, error) {
	if tenantID == "" {
		return 0, fmt.Errorf("tenant is required")
	}
	index := c.indexKey(tenantID, kind, value)
	if index == "" {
		return 0, fmt.Errorf("invalid invalidation kind %q", kind)
	}
	keys, err := c.client.SMembers(ctx, index).Result()
	if err != nil {
		return 0, fmt.Errorf("read invalidation index: %w", err)
	}
	if len(keys) == 0 {
		_ = c.client.Del(ctx, index).Err()
		return 0, nil
	}
	pipe := c.client.TxPipeline()
	deleteCmd := pipe.Del(ctx, keys...)
	pipe.Del(ctx, index)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("invalidate exact cache: %w", err)
	}
	return deleteCmd.Val(), nil
}

func (c *RedisExactCache) indexKey(tenantID, kind, value string) string {
	switch kind {
	case "tenant":
		return fmt.Sprintf("t:%s:cache:idx:all", tenantID)
	case "all":
		return fmt.Sprintf("t:%s:cache:idx:all", tenantID)
	case "model":
		if value == "" {
			return ""
		}
		return fmt.Sprintf("t:%s:cache:idx:model:%s", tenantID, value)
	case "provider":
		if value == "" {
			return ""
		}
		return fmt.Sprintf("t:%s:cache:idx:provider:%s", tenantID, value)
	case "prompt":
		if value == "" {
			return ""
		}
		return fmt.Sprintf("t:%s:cache:idx:prompt:%s", tenantID, value)
	default:
		return ""
	}
}
