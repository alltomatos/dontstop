package proxy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/proxy"
)

type fakeRepo struct {
	assignment   proxy.Endpoint
	assigned     bool
	candidates   []proxy.Endpoint
	assignCalls  []string
	failureCalls []string
	successCalls []string
	banCalls     []string
}

func (r *fakeRepo) Assignment(context.Context, string, string) (proxy.Endpoint, bool, error) {
	return r.assignment, r.assigned, nil
}

func (r *fakeRepo) Candidates(context.Context, string, string, string, int) ([]proxy.Endpoint, error) {
	return r.candidates, nil
}

func (r *fakeRepo) Assign(_ context.Context, _, _, endpointID string) error {
	r.assignCalls = append(r.assignCalls, endpointID)
	r.assigned = true
	r.assignment = proxy.Endpoint{ID: endpointID, Address: "203.0.113.20", Port: 8080, Valid: true}
	return nil
}

func (r *fakeRepo) RecordFailure(_ context.Context, _, endpointID string) error {
	r.failureCalls = append(r.failureCalls, endpointID)
	return nil
}

func (r *fakeRepo) RecordSuccess(_ context.Context, _, endpointID string) error {
	r.successCalls = append(r.successCalls, endpointID)
	return nil
}

func (r *fakeRepo) RecordBan(_ context.Context, _, _, endpointID, reason string, _ time.Time) error {
	r.banCalls = append(r.banCalls, endpointID+":"+reason)
	return nil
}

type fakeState struct {
	quarantined map[string]bool
	marked      []string
}

func (s *fakeState) Quarantine(_ context.Context, _, endpointID string, _ time.Duration) error {
	s.marked = append(s.marked, endpointID)
	return nil
}

func (s *fakeState) Quarantined(_ context.Context, _, endpointID string) (bool, error) {
	return s.quarantined[endpointID], nil
}

func (s *fakeState) ClearFailures(context.Context, string, string) error { return nil }

type fakeDecrypter struct{}

func (fakeDecrypter) Decrypt([]byte) ([]byte, error) { return []byte("secret"), nil }

func endpoint(id string) proxy.Endpoint {
	return proxy.Endpoint{
		ID: id, Address: "203.0.113.10", Port: 8080, Valid: true,
		UsernameCipher: []byte("user"), PasswordCipher: []byte("pass"), Score: 100,
	}
}

func TestResolveOffNeverAcquiresProxy(t *testing.T) {
	repo := &fakeRepo{assigned: true, assignment: endpoint("proxy-1")}
	coordinator := proxy.NewCoordinator(repo, &fakeState{}, fakeDecrypter{})
	lease, err := coordinator.Resolve(context.Background(), "tenant-1", "account-1", "off")
	if err != nil || lease.EndpointID != "" || lease.ProxyURL != "" {
		t.Fatalf("off must not acquire a proxy: %+v err=%v", lease, err)
	}
	if len(repo.assignCalls) != 0 {
		t.Fatalf("off must not assign: %v", repo.assignCalls)
	}
}

func TestResolveUsesHealthyAssignment(t *testing.T) {
	repo := &fakeRepo{assigned: true, assignment: endpoint("proxy-1")}
	coordinator := proxy.NewCoordinator(repo, &fakeState{}, fakeDecrypter{})
	lease, err := coordinator.Resolve(context.Background(), "tenant-1", "account-1", "preferred")
	if err != nil || lease.EndpointID != "proxy-1" || lease.ProxyURL == "" {
		t.Fatalf("unexpected lease: %+v err=%v", lease, err)
	}
}

func TestResolveSkipsRedisQuarantineAndPicksNext(t *testing.T) {
	repo := &fakeRepo{
		assigned: true, assignment: endpoint("proxy-1"),
		candidates: []proxy.Endpoint{endpoint("proxy-2")},
	}
	state := &fakeState{quarantined: map[string]bool{"proxy-1": true}}
	coordinator := proxy.NewCoordinator(repo, state, fakeDecrypter{})
	lease, err := coordinator.Resolve(context.Background(), "tenant-1", "account-1", "required")
	if err != nil || lease.EndpointID != "proxy-2" {
		t.Fatalf("expected next proxy, got %+v err=%v", lease, err)
	}
	if len(repo.assignCalls) != 1 || repo.assignCalls[0] != "proxy-2" {
		t.Fatalf("expected assignment to proxy-2, got %v", repo.assignCalls)
	}
}

func TestResolveRequiredFailsClosedWithoutPool(t *testing.T) {
	coordinator := proxy.NewCoordinator(&fakeRepo{}, &fakeState{}, fakeDecrypter{})
	_, err := coordinator.Resolve(context.Background(), "tenant-1", "account-1", "required")
	if !errors.Is(err, provider.ErrProxyRequired) {
		t.Fatalf("expected ErrProxyRequired, got %v", err)
	}
}

func TestResolvePreferredFallsBackDirectWithoutPool(t *testing.T) {
	coordinator := proxy.NewCoordinator(&fakeRepo{}, &fakeState{}, fakeDecrypter{})
	lease, err := coordinator.Resolve(context.Background(), "tenant-1", "account-1", "preferred")
	if err != nil || lease.EndpointID != "" {
		t.Fatalf("expected direct fallback, got %+v err=%v", lease, err)
	}
}

func TestRotateQuarantinesAndAssignsNext(t *testing.T) {
	repo := &fakeRepo{candidates: []proxy.Endpoint{endpoint("proxy-2")}}
	state := &fakeState{}
	coordinator := proxy.NewCoordinator(repo, state, fakeDecrypter{})
	lease, err := coordinator.Rotate(context.Background(), "tenant-1", "account-1", "proxy-1", "preferred", provider.ProxyFailureUnreachable, 15*time.Minute)
	if err != nil || lease.EndpointID != "proxy-2" {
		t.Fatalf("unexpected rotation: %+v err=%v", lease, err)
	}
	if len(repo.failureCalls) != 1 || repo.failureCalls[0] != "proxy-1" {
		t.Fatalf("expected failure on proxy-1, got %v", repo.failureCalls)
	}
	if len(state.marked) != 1 || state.marked[0] != "proxy-1" {
		t.Fatalf("expected redis quarantine on proxy-1, got %v", state.marked)
	}
}

func TestRotateRequiredFailsClosedWhenPoolExhausted(t *testing.T) {
	coordinator := proxy.NewCoordinator(&fakeRepo{}, &fakeState{}, fakeDecrypter{})
	_, err := coordinator.Rotate(context.Background(), "tenant-1", "account-1", "proxy-1", "required", provider.ProxyFailureUnreachable, 15*time.Minute)
	if !errors.Is(err, provider.ErrProxyRequired) {
		t.Fatalf("expected ErrProxyRequired, got %v", err)
	}
}

type fakeRefresher struct{ calls chan string }

func (f *fakeRefresher) RefreshPool(_ context.Context, tenantID string) error {
	f.calls <- tenantID
	return nil
}

func TestRotateTriggersPoolRefresh(t *testing.T) {
	repo := &fakeRepo{candidates: []proxy.Endpoint{endpoint("proxy-2")}}
	refresher := &fakeRefresher{calls: make(chan string, 1)}
	coordinator := proxy.NewCoordinator(repo, &fakeState{}, fakeDecrypter{})
	coordinator.SetPoolRefresher(refresher)
	if _, err := coordinator.Rotate(context.Background(), "tenant-1", "account-1", "proxy-1", "auto", provider.ProxyFailureBan, time.Hour); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	select {
	case tenantID := <-refresher.calls:
		if tenantID != "tenant-1" {
			t.Fatalf("unexpected refresh tenant: %s", tenantID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pool refresh was not triggered")
	}
}
