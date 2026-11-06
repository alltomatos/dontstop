package routing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// FailureKind drives cooldown duration and observability.
type FailureKind string

const (
	FailureAuth      FailureKind = "auth"
	FailureRateLimit FailureKind = "rate_limit"
	FailureServer    FailureKind = "server"
	FailureUnknown   FailureKind = "unknown"
)

var (
	ErrNoCandidate         = errors.New("no eligible account")
	ErrAllCandidatesFailed = errors.New("all eligible accounts failed")
)

// Candidate is an account that can serve a model request.
type Candidate struct {
	TenantID             string
	AccountID            string
	ProviderID           string
	ProviderName         string
	ProviderKind         string
	Priority             int
	Weight               int
	BaseURL              string
	CredentialCipher     []byte
	CredentialKeyVersion int
}

// Repository lists candidates for a tenant/model.
type Repository interface {
	Candidates(ctx context.Context, tenantID, model string) ([]Candidate, error)
}

// Breaker coordinates cooldown and circuit-breaker state.
type Breaker interface {
	Allow(ctx context.Context, account Candidate) (bool, error)
	Failure(ctx context.Context, account Candidate, kind FailureKind) error
	Success(ctx context.Context, account Candidate) error
}

// AttemptResult reports whether the upstream attempt already sent bytes.
type AttemptResult struct {
	FirstTokenSent bool
	Failure        FailureKind
}

// Attempt performs one upstream call for a selected candidate.
type Attempt func(ctx context.Context, account Candidate) (AttemptResult, error)

// Router selects accounts and fails over before the first token.
type Router struct {
	repository  Repository
	breaker     Breaker
	random      *rand.Rand
	maxAttempts int
}

// NewRouter creates an account router.
func NewRouter(repository Repository, breaker Breaker) *Router {
	return &Router{
		repository:  repository,
		breaker:     breaker,
		random:      rand.New(rand.NewSource(time.Now().UnixNano())),
		maxAttempts: 3,
	}
}

// SetMaxAttempts bounds upstream attempts per request.
func (r *Router) SetMaxAttempts(max int) {
	if max > 0 {
		r.maxAttempts = max
	}
}

// SetRandomSource makes weighted selection deterministic in tests.
func (r *Router) SetRandomSource(source rand.Source) {
	r.random = rand.New(source)
}

// Route selects a candidate and fails over until success or the first token.
func (r *Router) Route(ctx context.Context, tenantID, model string, attempt Attempt) (Candidate, error) {
	candidates, err := r.repository.Candidates(ctx, tenantID, model)
	if err != nil {
		return Candidate{}, fmt.Errorf("list routing candidates: %w", err)
	}
	if len(candidates) == 0 {
		return Candidate{}, ErrNoCandidate
	}

	ordered := r.order(candidates)
	var lastErr error
	attempts := 0
	for _, candidate := range ordered {
		if attempts >= r.maxAttempts {
			break
		}
		if err := ctx.Err(); err != nil {
			return Candidate{}, err
		}
		allowed, err := r.breaker.Allow(ctx, candidate)
		if err == nil && !allowed {
			continue
		}

		attempts++
		result, err := attempt(ctx, candidate)
		if err == nil {
			_ = r.breaker.Success(ctx, candidate)
			return candidate, nil
		}
		lastErr = err
		_ = r.breaker.Failure(ctx, candidate, result.Failure)
		if result.FirstTokenSent {
			return candidate, err
		}
	}

	if lastErr != nil {
		return Candidate{}, fmt.Errorf("%w: %w", ErrAllCandidatesFailed, lastErr)
	}
	return Candidate{}, ErrAllCandidatesFailed
}

func (r *Router) order(candidates []Candidate) []Candidate {
	ordered := append([]Candidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Priority < ordered[j].Priority
	})

	start := 0
	for start < len(ordered) {
		end := start + 1
		for end < len(ordered) && ordered[end].Priority == ordered[start].Priority {
			end++
		}
		r.shuffleWeighted(ordered[start:end])
		start = end
	}
	return ordered
}

func (r *Router) shuffleWeighted(group []Candidate) {
	remaining := append([]Candidate(nil), group...)
	for index := 0; index < len(group); index++ {
		total := 0
		for _, candidate := range remaining {
			weight := candidate.Weight
			if weight < 1 {
				weight = 1
			}
			total += weight
		}
		pick := r.random.Intn(total)
		selected := 0
		for selected < len(remaining) {
			weight := remaining[selected].Weight
			if weight < 1 {
				weight = 1
			}
			if pick < weight {
				break
			}
			pick -= weight
			selected++
		}
		group[index] = remaining[selected]
		remaining = append(remaining[:selected], remaining[selected+1:]...)
	}
}
