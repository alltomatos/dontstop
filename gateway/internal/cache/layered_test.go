package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

type stubL2 struct {
	entry      cache.Entry
	found      bool
	gets       int
	sets       int
	invalidates int
	setErr      error
	invalidate error
}

func (s *stubL2) Get(context.Context, string) (cache.Entry, bool, error) {
	s.gets++
	return s.entry, s.found, nil
}

func (s *stubL2) Set(context.Context, string, cache.Meta, cache.Entry, time.Duration) error {
	s.sets++
	return s.setErr
}

func (s *stubL2) Invalidate(context.Context, string, string, string) (int64, error) {
	s.invalidates++
	return 1, s.invalidate
}

func key(tenant string) string { return "t:" + tenant + ":cache:exact:abc" }

func TestLayeredCachePromotesL2HitToL1(t *testing.T) {
	l2 := &stubL2{found: true, entry: cache.Entry{Response: provider.ChatResponse{ID: "cached"}}}
	layered := cache.NewLayeredCache(l2, nil, cache.LayeredOptions{TTL: time.Minute})

	first, found, err := layered.Get(context.Background(), key("t1"))
	if err != nil || !found || first.Response.ID != "cached" {
		t.Fatalf("first get: found=%v err=%v entry=%+v", found, err, first)
	}
	second, found, err := layered.Get(context.Background(), key("t1"))
	if err != nil || !found || second.Response.ID != "cached" {
		t.Fatalf("second get: found=%v err=%v", found, err)
	}
	if l2.gets != 1 {
		t.Fatalf("L2 must be read once, got %d reads", l2.gets)
	}
}

func TestLayeredCacheExpiresL1(t *testing.T) {
	l2 := &stubL2{found: true, entry: cache.Entry{Response: provider.ChatResponse{ID: "cached"}}}
	layered := cache.NewLayeredCache(l2, nil, cache.LayeredOptions{TTL: 20 * time.Millisecond})

	if _, found, _ := layered.Get(context.Background(), key("t1")); !found {
		t.Fatal("expected first get to hit")
	}
	time.Sleep(30 * time.Millisecond)
	if _, found, _ := layered.Get(context.Background(), key("t1")); !found {
		t.Fatal("expected expired L1 to fall back to L2")
	}
	if l2.gets != 2 {
		t.Fatalf("expired L1 must re-read L2, got %d reads", l2.gets)
	}
}

func TestLayeredCacheSetRequiresL2Success(t *testing.T) {
	l2 := &stubL2{setErr: context.DeadlineExceeded}
	layered := cache.NewLayeredCache(l2, nil, cache.LayeredOptions{TTL: time.Minute})

	if err := layered.Set(context.Background(), key("t1"), cache.Meta{TenantID: "t1"}, cache.Entry{}, time.Hour); err == nil {
		t.Fatal("expected L2 error to surface")
	}
	l2.setErr = nil
	l2.found = false
	if _, found, _ := layered.Get(context.Background(), key("t1")); found {
		t.Fatal("L1 must not hold an entry that L2 rejected")
	}
}

func TestLayeredCacheInvalidateClearsOnlyTenant(t *testing.T) {
	l2 := &stubL2{}
	layered := cache.NewLayeredCache(l2, nil, cache.LayeredOptions{TTL: time.Minute})
	entry := cache.Entry{Response: provider.ChatResponse{ID: "x"}}
	_ = layered.Set(context.Background(), key("t1"), cache.Meta{TenantID: "t1"}, entry, time.Hour)
	_ = layered.Set(context.Background(), key("t2"), cache.Meta{TenantID: "t2"}, entry, time.Hour)

	// Warm L1 for both tenants by reading through.
	_, _, _ = layered.Get(context.Background(), key("t1"))
	_, _, _ = layered.Get(context.Background(), key("t2"))

	if _, err := layered.Invalidate(context.Background(), "t1", "tenant", ""); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	before := l2.gets
	if _, found, _ := layered.Get(context.Background(), key("t1")); found {
		t.Fatal("t1 entry must be dropped from L1")
	}
	if l2.gets == before {
		t.Fatal("t1 must go back to L2 after invalidation")
	}
	if _, found, _ := layered.Get(context.Background(), key("t2")); !found {
		t.Fatal("t2 entry must survive t1 invalidation")
	}
}

func TestLayeredCacheDisabledWithoutTTL(t *testing.T) {
	l2 := &stubL2{}
	if got := cache.NewLayeredCache(l2, nil, cache.LayeredOptions{}); got != l2 {
		t.Fatal("a non-positive TTL must return the L2 cache unchanged")
	}
}
