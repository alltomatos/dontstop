package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/redis/go-redis/v9"
)

// RedisIdempotencyStore stores non-streaming responses in Redis.
type RedisIdempotencyStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

// NewRedisIdempotencyStore creates an idempotency store with a bounded TTL.
func NewRedisIdempotencyStore(client redis.UniversalClient, ttl time.Duration) *RedisIdempotencyStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &RedisIdempotencyStore{client: client, ttl: ttl}
}

// Get returns a cached response for an idempotency key.
func (s *RedisIdempotencyStore) Get(ctx context.Context, key string) (provider.ChatResponse, bool, error) {
	raw, err := s.client.Get(ctx, "t:"+key).Bytes()
	if err == redis.Nil {
		return provider.ChatResponse{}, false, nil
	}
	if err != nil {
		return provider.ChatResponse{}, false, fmt.Errorf("read idempotency response: %w", err)
	}
	var response provider.ChatResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return provider.ChatResponse{}, false, fmt.Errorf("decode idempotency response: %w", err)
	}
	return response, true, nil
}

// Set stores a response for an idempotency key.
func (s *RedisIdempotencyStore) Set(ctx context.Context, key string, response provider.ChatResponse) error {
	raw, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode idempotency response: %w", err)
	}
	if err := s.client.Set(ctx, "t:"+key, raw, s.ttl).Err(); err != nil {
		return fmt.Errorf("store idempotency response: %w", err)
	}
	return nil
}
