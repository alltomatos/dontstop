package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type targetRepository struct {
	candidates map[string][]routing.Candidate
}

func (r targetRepository) Candidates(context.Context, string, string) ([]routing.Candidate, error) {
	return nil, nil
}

func (r targetRepository) CandidatesForTarget(_ context.Context, _, providerID, model string) ([]routing.Candidate, error) {
	return r.candidates[providerID+"/"+model], nil
}

func targetFixture(count int) (targetRepository, []routing.ComboTarget) {
	repository := targetRepository{candidates: map[string][]routing.Candidate{}}
	targets := make([]routing.ComboTarget, 0, count)
	for index := 0; index < count; index++ {
		providerID := "provider-" + string(rune('a'+index))
		model := "model-" + string(rune('a'+index))
		repository.candidates[providerID+"/"+model] = []routing.Candidate{{
			AccountID: "account-" + string(rune('a'+index)), ProviderID: providerID, Priority: index, Weight: 1,
		}}
		targets = append(targets, routing.ComboTarget{ProviderID: providerID, ModelID: model, Priority: index, Position: index, Weight: 1})
	}
	return repository, targets
}

func allowAll(t *testing.T, targets int) *recordingBreaker {
	t.Helper()
	breaker := newRecordingBreaker()
	for index := 0; index < targets; index++ {
		breaker.allowed["account-"+string(rune('a'+index))] = true
	}
	return breaker
}

func TestRouteTargetsHonorsPerComboAttemptBudget(t *testing.T) {
	repository, targets := targetFixture(6)
	router := routing.NewRouter(repository, allowAll(t, 6))
	router.SetMaxAttempts(8)

	attempts := 0
	candidate, err := router.RouteTargets(context.Background(), "tenant-1", targets, 5, func(context.Context, routing.Candidate) (routing.AttemptResult, error) {
		attempts++
		if attempts < 5 {
			return routing.AttemptResult{Failure: routing.FailureServer}, errors.New("upstream failed")
		}
		return routing.AttemptResult{}, nil
	})
	if err != nil {
		t.Fatalf("expected success on the fifth attempt: %v", err)
	}
	if attempts != 5 || candidate.AccountID != "account-e" {
		t.Fatalf("unexpected attempts/candidate: %d %+v", attempts, candidate)
	}
}

func TestRouteTargetsTightensToRouterCeiling(t *testing.T) {
	repository, targets := targetFixture(6)
	router := routing.NewRouter(repository, allowAll(t, 6))
	router.SetMaxAttempts(3)

	attempts := 0
	_, err := router.RouteTargets(context.Background(), "tenant-1", targets, 8, func(context.Context, routing.Candidate) (routing.AttemptResult, error) {
		attempts++
		return routing.AttemptResult{Failure: routing.FailureServer}, errors.New("upstream failed")
	})
	if !errors.Is(err, routing.ErrAllCandidatesFailed) || attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d err=%v", attempts, err)
	}
}

func TestRouteTargetsComboBudgetCanShrink(t *testing.T) {
	repository, targets := targetFixture(6)
	router := routing.NewRouter(repository, allowAll(t, 6))
	router.SetMaxAttempts(8)

	attempts := 0
	_, err := router.RouteTargets(context.Background(), "tenant-1", targets, 2, func(context.Context, routing.Candidate) (routing.AttemptResult, error) {
		attempts++
		return routing.AttemptResult{Failure: routing.FailureServer}, errors.New("upstream failed")
	})
	if !errors.Is(err, routing.ErrAllCandidatesFailed) || attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d err=%v", attempts, err)
	}
}
