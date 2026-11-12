package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type staticRepository struct {
	candidates []routing.Candidate
}

func (r staticRepository) Candidates(context.Context, string, string) ([]routing.Candidate, error) {
	return r.candidates, nil
}

type recordingBreaker struct {
	allowed  map[string]bool
	failures map[string]routing.FailureKind
	success  map[string]bool
}

func newRecordingBreaker() *recordingBreaker {
	return &recordingBreaker{
		allowed:  make(map[string]bool),
		failures: make(map[string]routing.FailureKind),
		success:  make(map[string]bool),
	}
}

func (b *recordingBreaker) Allow(_ context.Context, account routing.Candidate) (bool, error) {
	return b.allowed[account.AccountID], nil
}

func (b *recordingBreaker) Failure(_ context.Context, account routing.Candidate, kind routing.FailureKind) error {
	b.failures[account.AccountID] = kind
	return nil
}

func (b *recordingBreaker) Success(_ context.Context, account routing.Candidate) error {
	b.success[account.AccountID] = true
	return nil
}

func TestRouteFailsOverBeforeFirstToken(t *testing.T) {
	breaker := newRecordingBreaker()
	breaker.allowed["account-a"] = true
	breaker.allowed["account-b"] = true
	router := routing.NewRouter(staticRepository{candidates: []routing.Candidate{
		{AccountID: "account-a", ProviderID: "provider-a", Priority: 10, Weight: 1},
		{AccountID: "account-b", ProviderID: "provider-b", Priority: 20, Weight: 1},
	}}, breaker)

	var attempts []string
	_, err := router.Route(context.Background(), "tenant-1", "gpt-4.1", func(_ context.Context, account routing.Candidate) (routing.AttemptResult, error) {
		attempts = append(attempts, account.AccountID)
		if account.AccountID == "account-a" {
			return routing.AttemptResult{Failure: routing.FailureRateLimit}, errors.New("429")
		}
		return routing.AttemptResult{}, nil
	})
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if len(attempts) != 2 || attempts[0] != "account-a" || attempts[1] != "account-b" {
		t.Fatalf("unexpected attempts: %+v", attempts)
	}
	if breaker.failures["account-a"] != routing.FailureRateLimit {
		t.Fatalf("expected rate-limit feedback, got %v", breaker.failures["account-a"])
	}
	if !breaker.success["account-b"] {
		t.Fatal("expected success feedback for account-b")
	}
}

func TestRouteStopsAfterFirstToken(t *testing.T) {
	breaker := newRecordingBreaker()
	breaker.allowed["account-a"] = true
	breaker.allowed["account-b"] = true
	router := routing.NewRouter(staticRepository{candidates: []routing.Candidate{
		{AccountID: "account-a", Priority: 10, Weight: 1},
		{AccountID: "account-b", Priority: 20, Weight: 1},
	}}, breaker)

	calls := 0
	_, err := router.Route(context.Background(), "tenant-1", "gpt-4.1", func(context.Context, routing.Candidate) (routing.AttemptResult, error) {
		calls++
		return routing.AttemptResult{FirstTokenSent: true, Failure: routing.FailureServer}, errors.New("stream broke")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("must not retry after first token, got %d calls", calls)
	}
}

func TestRouteSkipsOpenAccounts(t *testing.T) {
	breaker := newRecordingBreaker()
	breaker.allowed["account-a"] = false
	breaker.allowed["account-b"] = true
	router := routing.NewRouter(staticRepository{candidates: []routing.Candidate{
		{AccountID: "account-a", Priority: 10, Weight: 1},
		{AccountID: "account-b", Priority: 20, Weight: 1},
	}}, breaker)

	called := ""
	_, err := router.Route(context.Background(), "tenant-1", "gpt-4.1", func(_ context.Context, account routing.Candidate) (routing.AttemptResult, error) {
		called = account.AccountID
		return routing.AttemptResult{}, nil
	})
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if called != "account-b" {
		t.Fatalf("expected account-b, got %q", called)
	}
}

func TestRouteReturnsErrorWhenNoAccountIsAvailable(t *testing.T) {
	router := routing.NewRouter(staticRepository{}, newRecordingBreaker())
	if _, err := router.Route(context.Background(), "tenant-1", "gpt-4.1", func(context.Context, routing.Candidate) (routing.AttemptResult, error) {
		return routing.AttemptResult{}, nil
	}); !errors.Is(err, routing.ErrNoCandidate) {
		t.Fatalf("expected ErrNoCandidate, got %v", err)
	}
}

func TestRouteFailsOverWithinProviderBeforeAnotherProvider(t *testing.T) {
	breaker := newRecordingBreaker()
	for _, id := range []string{"account-a1", "account-a2", "account-b1"} {
		breaker.allowed[id] = true
	}
	router := routing.NewRouter(staticRepository{candidates: []routing.Candidate{
		{AccountID: "account-a1", ProviderID: "provider-a", Priority: 10, Weight: 1},
		{AccountID: "account-b1", ProviderID: "provider-b", Priority: 15, Weight: 1},
		{AccountID: "account-a2", ProviderID: "provider-a", Priority: 20, Weight: 1},
	}}, breaker)

	var attempts []string
	_, err := router.Route(context.Background(), "tenant-1", "gpt-4.1", func(_ context.Context, account routing.Candidate) (routing.AttemptResult, error) {
		attempts = append(attempts, account.AccountID)
		switch account.AccountID {
		case "account-a1":
			return routing.AttemptResult{Failure: routing.FailureRateLimit}, errors.New("429")
		case "account-a2":
			return routing.AttemptResult{Failure: routing.FailureQuota}, errors.New("insufficient_quota")
		default:
			return routing.AttemptResult{}, nil
		}
	})
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	want := []string{"account-a1", "account-a2", "account-b1"}
	if len(attempts) != len(want) {
		t.Fatalf("unexpected attempts: %v", attempts)
	}
	for index := range want {
		if attempts[index] != want[index] {
			t.Fatalf("unexpected order: %v", attempts)
		}
	}
	if breaker.failures["account-a1"] != routing.FailureRateLimit || breaker.failures["account-a2"] != routing.FailureQuota {
		t.Fatalf("expected reason-specific cooldowns: %+v", breaker.failures)
	}
}

func TestRouteStopsOnTerminalClientError(t *testing.T) {
	breaker := newRecordingBreaker()
	breaker.allowed["account-a"] = true
	breaker.allowed["account-b"] = true
	router := routing.NewRouter(staticRepository{candidates: []routing.Candidate{
		{AccountID: "account-a", ProviderID: "provider-a", Priority: 10, Weight: 1},
		{AccountID: "account-b", ProviderID: "provider-a", Priority: 20, Weight: 1},
	}}, breaker)

	calls := 0
	_, err := router.Route(context.Background(), "tenant-1", "gpt-4.1", func(context.Context, routing.Candidate) (routing.AttemptResult, error) {
		calls++
		return routing.AttemptResult{Failure: routing.FailureTerminal}, errors.New("invalid request")
	})
	if err == nil {
		t.Fatal("expected terminal error")
	}
	if calls != 1 {
		t.Fatalf("client errors must not consume another account, got %d calls", calls)
	}
	if len(breaker.failures) != 0 {
		t.Fatalf("client errors must not open the breaker: %+v", breaker.failures)
	}
}
