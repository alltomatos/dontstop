package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LayeredCache places a bounded in-process L1 in front of a shared L2 cache.
//
// L1 exists to absorb the bursts of identical requests that happen inside a
// single gateway replica; it is deliberately short-lived (seconds, not hours)
// so a stale entry can never outlive its TTL. Invalidation clears the local
// replica immediately and is broadcast to the others through Redis pub/sub, so
// no replica keeps serving an entry that was invalidated elsewhere.
type LayeredCache struct {
	l2           ExactCache
	pubsub       redis.UniversalClient
	local        *localCache
	invalidateCh string
}

// LayeredOptions configures the L1 layer.
type LayeredOptions struct {
	// TTL is the L1 entry lifetime. Zero or negative disables L1.
	TTL time.Duration
	// MaxEntries bounds the number of L1 entries; zero uses a safe default.
	MaxEntries int
}

const defaultL1Entries = 2048

// NewLayeredCache wraps an L2 cache with an L1 layer. A nil or non-positive TTL
// returns the L2 cache unchanged so the call site never needs a special case.
func NewLayeredCache(l2 ExactCache, pubsub redis.UniversalClient, opts LayeredOptions) ExactCache {
	if l2 == nil || opts.TTL <= 0 {
		return l2
	}
	maxEntries := opts.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultL1Entries
	}
	return &LayeredCache{
		l2:           l2,
		pubsub:       pubsub,
		local:        newLocalCache(opts.TTL, maxEntries),
		invalidateCh: "dontstop:cache:invalidate",
	}
}

// Get returns a cached entry from L1, falling back to L2 and promoting on hit.
func (c *LayeredCache) Get(ctx context.Context, key string) (Entry, bool, error) {
	if entry, ok := c.local.get(key); ok {
		return entry, true, nil
	}
	entry, found, err := c.l2.Get(ctx, key)
	if err != nil || !found {
		return entry, found, err
	}
	c.local.set(key, entry)
	return entry, true, nil
}

// Set writes the entry to L2 first. L1 is only populated after L2 accepts it, so
// an entry that failed to persist upstream is never served from memory alone.
func (c *LayeredCache) Set(ctx context.Context, key string, meta Meta, entry Entry, ttl time.Duration) error {
	if err := c.l2.Set(ctx, key, meta, entry, ttl); err != nil {
		return err
	}
	c.local.set(key, entry)
	return nil
}

// Invalidate clears the matching L1 entries, invalidates L2 and broadcasts the
// invalidation so the other replicas drop their own L1 copies.
func (c *LayeredCache) Invalidate(ctx context.Context, tenantID, kind, value string) (int64, error) {
	removed, err := c.l2.Invalidate(ctx, tenantID, kind, value)
	if err != nil {
		return removed, err
	}
	c.local.clearTenant(tenantID)
	if c.pubsub != nil {
		payload, marshalErr := json.Marshal(invalidateMessage{TenantID: tenantID, Kind: kind, Value: value})
		if marshalErr == nil {
			_ = c.pubsub.Publish(ctx, c.invalidateCh, payload).Err()
		}
	}
	return removed, nil
}

type invalidateMessage struct {
	TenantID string `json:"tenant_id"`
	Kind     string `json:"kind"`
	Value    string `json:"value"`
}

// RunSubscription listens for cross-replica invalidations until the context is
// cancelled. It is safe to run in a goroutine and returns early when no Redis
// pub/sub client is configured.
func (c *LayeredCache) RunSubscription(ctx context.Context, onError func(error)) {
	if c.pubsub == nil {
		return
	}
	sub := c.pubsub.Subscribe(ctx, c.invalidateCh)
	defer func() { _ = sub.Close() }()
	channel := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-channel:
			if !ok {
				return
			}
			var payload invalidateMessage
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				if onError != nil {
					onError(fmt.Errorf("decode cache invalidation: %w", err))
				}
				continue
			}
			c.local.clearTenant(payload.TenantID)
		}
	}
}

// localCache is a tiny TTL + size bounded cache with no eviction ordering beyond
// dropping expired entries during writes.
type localCache struct {
	mu        sync.RWMutex
	entries    map[string]localEntry
	ttl        time.Duration
	maxEntries int
}

type localEntry struct {
	entry   Entry
	expires time.Time
}

func newLocalCache(ttl time.Duration, maxEntries int) *localCache {
	return &localCache{entries: make(map[string]localEntry), ttl: ttl, maxEntries: maxEntries}
}

func (c *localCache) get(key string) (Entry, bool) {
	c.mu.RLock()
	item, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return Entry{}, false
	}
	if time.Now().After(item.expires) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return Entry{}, false
	}
	return item.entry, true
}

func (c *localCache) set(key string, entry Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxEntries {
		now := time.Now()
		for existing, item := range c.entries {
			if now.After(item.expires) {
				delete(c.entries, existing)
			}
		}
		if len(c.entries) >= c.maxEntries {
			for existing := range c.entries {
				delete(c.entries, existing)
				break
			}
		}
	}
	c.entries[key] = localEntry{entry: entry, expires: time.Now().Add(c.ttl)}
}

func (c *localCache) clearTenant(tenantID string) {
	if tenantID == "" {
		return
	}
	prefix := "t:" + tenantID + ":"
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
}
