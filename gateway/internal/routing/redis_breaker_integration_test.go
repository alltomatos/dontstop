package routing_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/redis/go-redis/v9"
)

func TestRedisBreakerOpensHalfOpensAndCloses(t *testing.T) {
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
	ctx := context.Background()
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}

	breaker := routing.NewRedisBreaker(client, routing.BreakerConfig{
		BaseDelays: map[routing.FailureKind]time.Duration{
			routing.FailureRateLimit: 20 * time.Millisecond,
		},
		MaxDelay:   100 * time.Millisecond,
		FailureTTL: time.Minute,
		ProbeTTL:   time.Second,
	})
	account := routing.Candidate{TenantID: "tenant-1", AccountID: "account-a"}

	if allowed, err := breaker.Allow(ctx, account); err != nil || !allowed {
		t.Fatalf("closed breaker must allow: allowed=%v err=%v", allowed, err)
	}
	if err := breaker.Failure(ctx, account, routing.FailureRateLimit); err != nil {
		t.Fatalf("record failure: %v", err)
	}
	if allowed, err := breaker.Allow(ctx, account); err != nil || allowed {
		t.Fatalf("open breaker must block: allowed=%v err=%v", allowed, err)
	}
	if state, err := breaker.State(ctx, account); err != nil || state != "open" {
		t.Fatalf("expected open, got %q err=%v", state, err)
	}

	time.Sleep(30 * time.Millisecond)
	if state, err := breaker.State(ctx, account); err != nil || state != "half_open" {
		t.Fatalf("expected half_open, got %q err=%v", state, err)
	}
	if allowed, err := breaker.Allow(ctx, account); err != nil || !allowed {
		t.Fatalf("half-open probe must be allowed: allowed=%v err=%v", allowed, err)
	}
	if allowed, err := breaker.Allow(ctx, account); err != nil || allowed {
		t.Fatalf("second half-open probe must be blocked: allowed=%v err=%v", allowed, err)
	}

	if err := breaker.Success(ctx, account); err != nil {
		t.Fatalf("record success: %v", err)
	}
	if state, err := breaker.State(ctx, account); err != nil || state != "closed" {
		t.Fatalf("expected closed, got %q err=%v", state, err)
	}
}

func TestRedisBreakerIsTenantScoped(t *testing.T) {
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
	ctx := context.Background()
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
	breaker := routing.NewRedisBreaker(client, routing.BreakerConfig{
		BaseDelays: map[routing.FailureKind]time.Duration{routing.FailureRateLimit: time.Minute},
		MaxDelay:   time.Minute,
		FailureTTL: time.Minute,
		ProbeTTL:   time.Second,
	})
	tenantA := routing.Candidate{TenantID: "tenant-a", AccountID: "shared-id"}
	tenantB := routing.Candidate{TenantID: "tenant-b", AccountID: "shared-id"}

	if err := breaker.Failure(ctx, tenantA, routing.FailureRateLimit); err != nil {
		t.Fatalf("record tenant A failure: %v", err)
	}
	if allowed, err := breaker.Allow(ctx, tenantA); err != nil || allowed {
		t.Fatalf("tenant A must be blocked: allowed=%v err=%v", allowed, err)
	}
	if allowed, err := breaker.Allow(ctx, tenantB); err != nil || !allowed {
		t.Fatalf("tenant B must not inherit tenant A state: allowed=%v err=%v", allowed, err)
	}
}
