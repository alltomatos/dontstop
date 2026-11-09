package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	loginWindow           = 5 * time.Minute
	loginMaxAttempts      = 10
	loginMaxTrackedEmails = 10000
	loginRedisPrefix      = "dontstop:login:"
	loginRedisTimeout     = time.Second
)

var loginAllowScript = redis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
return redis.call('ZCARD', KEYS[1])
`)

type loginLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	window   time.Duration
	max      int
	attempts map[string][]time.Time
	redis    *redis.Client
	seq      uint64
}

func newLoginLimiter(clients ...*redis.Client) *loginLimiter {
	limiter := &loginLimiter{
		now:      time.Now,
		window:   loginWindow,
		max:      loginMaxAttempts,
		attempts: make(map[string][]time.Time),
	}
	if len(clients) > 0 {
		limiter.redis = clients[0]
	}
	return limiter
}

func (l *loginLimiter) Allow(email string) bool {
	if l.redis != nil {
		allowed, err := l.allowRedis(email)
		if err == nil {
			return allowed
		}
	}
	return l.allowMemory(email)
}

func (l *loginLimiter) Fail(email string) {
	if l.redis != nil {
		if err := l.failRedis(email); err == nil {
			return
		}
	}
	l.failMemory(email)
}

func (l *loginLimiter) Reset(email string) {
	if l.redis != nil {
		_ = l.resetRedis(email)
	}
	l.resetMemory(email)
}

func (l *loginLimiter) allowMemory(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[email] = pruneAttempts(l.attempts[email], l.now().Add(-l.window))
	if len(l.attempts[email]) == 0 {
		delete(l.attempts, email)
	}
	return len(l.attempts[email]) < l.max
}

func (l *loginLimiter) failMemory(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.attempts) >= loginMaxTrackedEmails {
		l.sweepLocked()
	}
	if len(l.attempts) >= loginMaxTrackedEmails {
		return
	}
	l.attempts[email] = append(l.attempts[email], l.now())
}

func (l *loginLimiter) resetMemory(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, email)
}

func (l *loginLimiter) allowRedis(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), loginRedisTimeout)
	defer cancel()
	now := l.now()
	count, err := loginAllowScript.Run(
		ctx,
		l.redis,
		[]string{l.redisKey(email)},
		now.Add(-l.window).UnixNano(),
	).Int64()
	if err != nil {
		return false, fmt.Errorf("check login limit: %w", err)
	}
	return count < int64(l.max), nil
}

func (l *loginLimiter) failRedis(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), loginRedisTimeout)
	defer cancel()
	key := l.redisKey(email)
	now := l.now().UnixNano()
	member := strconv.FormatInt(now, 10) + "-" + strconv.FormatUint(atomic.AddUint64(&l.seq, 1), 10)
	if err := l.redis.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: member}).Err(); err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	if err := l.redis.Expire(ctx, key, l.window+time.Minute).Err(); err != nil {
		return fmt.Errorf("expire login limit: %w", err)
	}
	return nil
}

func (l *loginLimiter) resetRedis(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), loginRedisTimeout)
	defer cancel()
	return l.redis.Del(ctx, l.redisKey(email)).Err()
}

func (l *loginLimiter) redisKey(email string) string {
	sum := sha256.Sum256([]byte(email))
	return loginRedisPrefix + hex.EncodeToString(sum[:])
}

func (l *loginLimiter) sweepLocked() {
	cutoff := l.now().Add(-l.window)
	for email, attempts := range l.attempts {
		pruned := pruneAttempts(attempts, cutoff)
		if len(pruned) == 0 {
			delete(l.attempts, email)
			continue
		}
		l.attempts[email] = pruned
	}
}

func pruneAttempts(attempts []time.Time, cutoff time.Time) []time.Time {
	kept := attempts[:0]
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	return kept
}
