package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// BreakerConfig controls cooldown and circuit-breaker timing.
type BreakerConfig struct {
	BaseDelays map[FailureKind]time.Duration
	MaxDelay   time.Duration
	FailureTTL time.Duration
	ProbeTTL   time.Duration
}

// RedisBreaker stores cooldown and half-open state in Redis.
type RedisBreaker struct {
	client redis.UniversalClient
	config BreakerConfig
}

// NewRedisBreaker creates a Redis-backed circuit breaker.
func NewRedisBreaker(client redis.UniversalClient, config BreakerConfig) *RedisBreaker {
	defaults := BreakerConfig{
		BaseDelays: map[FailureKind]time.Duration{
			FailureAuth:      5 * time.Minute,
			FailureQuota:     5 * time.Minute,
			FailureRateLimit: 30 * time.Second,
			FailureServer:    10 * time.Second,
			FailureProxy:     30 * time.Second,
			FailureUnknown:   5 * time.Second,
		},
		MaxDelay:   15 * time.Minute,
		FailureTTL: 24 * time.Hour,
		ProbeTTL:   10 * time.Second,
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = defaults.MaxDelay
	}
	if config.FailureTTL <= 0 {
		config.FailureTTL = defaults.FailureTTL
	}
	if config.ProbeTTL <= 0 {
		config.ProbeTTL = defaults.ProbeTTL
	}
	if config.BaseDelays == nil {
		config.BaseDelays = defaults.BaseDelays
	} else {
		for kind, delay := range defaults.BaseDelays {
			if config.BaseDelays[kind] <= 0 {
				config.BaseDelays[kind] = delay
			}
		}
	}
	return &RedisBreaker{client: client, config: config}
}

// Allow reports whether a request may use the account.
func (b *RedisBreaker) Allow(ctx context.Context, account Candidate) (bool, error) {
	open, err := b.client.Exists(ctx, b.key(account, "open")).Result()
	if err != nil {
		return true, fmt.Errorf("read breaker state: %w", err)
	}
	if open > 0 {
		return false, nil
	}
	failures, err := b.client.Exists(ctx, b.key(account, "failures")).Result()
	if err != nil {
		return true, fmt.Errorf("read breaker failures: %w", err)
	}
	if failures == 0 {
		return true, nil
	}
	probe, err := b.client.SetNX(ctx, b.key(account, "probe"), "1", b.config.ProbeTTL).Result()
	if err != nil {
		return true, fmt.Errorf("acquire half-open probe: %w", err)
	}
	return probe, nil
}

// Failure records a failed attempt and opens the breaker with exponential backoff.
func (b *RedisBreaker) Failure(ctx context.Context, account Candidate, kind FailureKind) error {
	failures, err := b.client.Incr(ctx, b.key(account, "failures")).Result()
	if err != nil {
		return fmt.Errorf("increment failures: %w", err)
	}
	if err := b.client.Expire(ctx, b.key(account, "failures"), b.config.FailureTTL).Err(); err != nil {
		return fmt.Errorf("expire failures: %w", err)
	}
	delay := b.delay(kind, failures)
	if err := b.client.Set(ctx, b.key(account, "open"), "1", delay).Err(); err != nil {
		return fmt.Errorf("open breaker: %w", err)
	}
	_ = b.client.Del(ctx, b.key(account, "probe")).Err()
	return nil
}

// Success closes the breaker and clears failure history.
func (b *RedisBreaker) Success(ctx context.Context, account Candidate) error {
	if err := b.client.Del(ctx, b.key(account, "failures"), b.key(account, "open"), b.key(account, "probe")).Err(); err != nil {
		return fmt.Errorf("close breaker: %w", err)
	}
	return nil
}

// State reports closed, open or half_open for observability and tests.
func (b *RedisBreaker) State(ctx context.Context, account Candidate) (string, error) {
	open, err := b.client.Exists(ctx, b.key(account, "open")).Result()
	if err != nil {
		return "", fmt.Errorf("read breaker state: %w", err)
	}
	if open > 0 {
		return "open", nil
	}
	failures, err := b.client.Exists(ctx, b.key(account, "failures")).Result()
	if err != nil {
		return "", fmt.Errorf("read breaker failures: %w", err)
	}
	if failures > 0 {
		return "half_open", nil
	}
	return "closed", nil
}

func (b *RedisBreaker) key(account Candidate, suffix string) string {
	return fmt.Sprintf("t:%s:cb:%s:%s", account.TenantID, account.AccountID, suffix)
}

func (b *RedisBreaker) delay(kind FailureKind, failures int64) time.Duration {
	base := b.config.BaseDelays[kind]
	if base <= 0 {
		base = b.config.BaseDelays[FailureUnknown]
	}
	delay := base
	for i := int64(1); i < failures && delay < b.config.MaxDelay; i++ {
		delay *= 2
	}
	if delay > b.config.MaxDelay {
		delay = b.config.MaxDelay
	}
	return delay
}
