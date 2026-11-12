package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisState stores transient proxy quarantine state per tenant.
type RedisState struct {
	client redis.UniversalClient
}

// NewRedisState creates a Redis-backed proxy state.
func NewRedisState(client redis.UniversalClient) *RedisState {
	return &RedisState{client: client}
}

// Quarantine blocks the endpoint for the given TTL.
func (s *RedisState) Quarantine(ctx context.Context, tenantID, endpointID string, ttl time.Duration) error {
	key := s.key(tenantID, "q", endpointID)
	if err := s.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("quarantine proxy: %w", err)
	}
	failures := s.key(tenantID, "fail", endpointID)
	if err := s.client.Incr(ctx, failures).Err(); err != nil {
		return fmt.Errorf("count proxy failure: %w", err)
	}
	if err := s.client.Expire(ctx, failures, time.Hour).Err(); err != nil {
		return fmt.Errorf("expire proxy failures: %w", err)
	}
	return nil
}

// Quarantined reports whether the endpoint is currently blocked.
func (s *RedisState) Quarantined(ctx context.Context, tenantID, endpointID string) (bool, error) {
	count, err := s.client.Exists(ctx, s.key(tenantID, "q", endpointID)).Result()
	if err != nil {
		return false, fmt.Errorf("read proxy quarantine: %w", err)
	}
	return count > 0, nil
}

// ClearFailures removes the consecutive failure counter after a success.
func (s *RedisState) ClearFailures(ctx context.Context, tenantID, endpointID string) error {
	if err := s.client.Del(ctx, s.key(tenantID, "fail", endpointID)).Err(); err != nil {
		return fmt.Errorf("clear proxy failures: %w", err)
	}
	return nil
}

func (s *RedisState) key(tenantID, kind, endpointID string) string {
	return fmt.Sprintf("t:%s:proxy:%s:%s", tenantID, kind, endpointID)
}
