package routing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// FailureKind drives cooldown duration and observability.
type FailureKind string

const (
	FailureAuth      FailureKind = "auth"
	FailureQuota     FailureKind = "quota"
	FailureRateLimit FailureKind = "rate_limit"
	FailureServer    FailureKind = "server"
	FailureProxy     FailureKind = "proxy"
	FailureTerminal  FailureKind = "terminal"
	FailureUnknown   FailureKind = "unknown"
)

var (
	ErrNoCandidate         = errors.New("no eligible account")
	ErrAllCandidatesFailed = errors.New("all eligible accounts failed")
)

var tracer = otel.Tracer("dontstop/gateway/routing")

// Candidate is an account that can serve a model request.
type Candidate struct {
	TenantID             string
	AccountID            string
	ProviderID           string
	ProviderName         string
	ProviderKind         string
	ModelID              string
	Priority             int
	Weight               int
	BaseURL              string
	CredentialCipher     []byte
	CredentialKeyVersion int
	ProxyMode            string
	ProxyEndpointID      string
	ProxyAddress         string
	ProxyPort            int
	ProxyCountry         string
	ProxyCity            string
	ProxyUsernameCipher  []byte
	ProxyPasswordCipher  []byte
	ProxyHealthy         bool
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
	ctx, span := tracer.Start(ctx, "routing.failover")
	defer span.End()
	span.SetAttributes(
		attribute.String("dontstop.tenant_id", tenantID),
		attribute.String("gen_ai.request.model", model),
	)
	candidates, err := r.repository.Candidates(ctx, tenantID, model)
	if err != nil {
		return Candidate{}, fmt.Errorf("list routing candidates: %w", err)
	}
	if len(candidates) == 0 {
		return Candidate{}, ErrNoCandidate
	}

	ordered := r.order(candidates)
	span.SetAttributes(attribute.String("routing.original_account_id", ordered[0].AccountID))
	var lastErr error
	attempts := 0
	failovers := 0
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
		attemptCtx, cancel := context.WithCancel(ctx)
		started := time.Now()
		result, err := attempt(attemptCtx, candidate)
		cancel()
		latency := time.Since(started)
		if err == nil {
			_ = r.breaker.Success(ctx, candidate)
			span.SetAttributes(
				attribute.String("routing.selected_account_id", candidate.AccountID),
				attribute.Int("routing.attempts", attempts),
				attribute.Int("routing.failovers", failovers),
			)
			return candidate, nil
		}
		lastErr = err
		span.AddEvent("routing.attempt_failed", trace.WithAttributes(
			attribute.String("routing.account_id", candidate.AccountID),
			attribute.String("routing.provider_id", candidate.ProviderID),
			attribute.String("routing.failure", string(result.Failure)),
			attribute.Int("routing.attempt", attempts),
			attribute.Int64("routing.latency_ms", latency.Milliseconds()),
		))
		if result.Failure == FailureTerminal {
			span.SetAttributes(attribute.String("routing.error_kind", string(result.Failure)))
			return candidate, err
		}
		_ = r.breaker.Failure(ctx, candidate, result.Failure)
		if result.FirstTokenSent {
			return candidate, err
		}
		failovers++
	}

	span.SetAttributes(
		attribute.Int("routing.attempts", attempts),
		attribute.Int("routing.failovers", failovers),
		attribute.String("routing.error_kind", "exhausted"),
	)
	if lastErr != nil {
		return Candidate{}, fmt.Errorf("%w: %w", ErrAllCandidatesFailed, lastErr)
	}
	return Candidate{}, ErrAllCandidatesFailed
}

// TargetRepository loads accounts for an explicit provider/model target.
type TargetRepository interface {
	CandidatesForTarget(ctx context.Context, tenantID, providerID, model string) ([]Candidate, error)
}

// RouteTargets selects a combo member and fails over across members/accounts.
func (r *Router) RouteTargets(ctx context.Context, tenantID string, targets []ComboTarget, attempt Attempt) (Candidate, error) {
	ctx, span := tracer.Start(ctx, "routing.combo")
	defer span.End()
	repository, ok := r.repository.(TargetRepository)
	if !ok {
		return Candidate{}, fmt.Errorf("routing repository does not support combo targets")
	}
	if len(targets) == 0 {
		return Candidate{}, ErrNoCandidate
	}
	var lastErr error
	attempts := 0
	failovers := 0
	for _, target := range targets {
		if attempts >= r.maxAttempts {
			break
		}
		candidates, err := repository.CandidatesForTarget(ctx, tenantID, target.ProviderID, target.ModelID)
		if err != nil {
			return Candidate{}, fmt.Errorf("list combo candidates: %w", err)
		}
		for _, candidate := range r.order(candidates) {
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
			attemptCtx, cancel := context.WithCancel(ctx)
			started := time.Now()
			result, err := attempt(attemptCtx, candidate)
			cancel()
			latency := time.Since(started)
			if err == nil {
				_ = r.breaker.Success(ctx, candidate)
				span.SetAttributes(
					attribute.String("routing.combo.provider_id", target.ProviderID),
					attribute.String("routing.combo.model_id", target.ModelID),
					attribute.Int("routing.attempts", attempts),
					attribute.Int("routing.failovers", failovers),
				)
				return candidate, nil
			}
			lastErr = err
			span.AddEvent("routing.combo_attempt_failed", trace.WithAttributes(
				attribute.String("routing.provider_id", candidate.ProviderID),
				attribute.String("routing.model_id", target.ModelID),
				attribute.String("routing.failure", string(result.Failure)),
				attribute.Int("routing.attempt", attempts),
				attribute.Int64("routing.latency_ms", latency.Milliseconds()),
			))
			if result.Failure == FailureTerminal {
				return candidate, err
			}
			_ = r.breaker.Failure(ctx, candidate, result.Failure)
			if result.FirstTokenSent {
				return candidate, err
			}
			failovers++
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
	order := make([]string, 0)
	groups := map[string][]Candidate{}
	for _, candidate := range ordered {
		if _, ok := groups[candidate.ProviderID]; !ok {
			order = append(order, candidate.ProviderID)
		}
		groups[candidate.ProviderID] = append(groups[candidate.ProviderID], candidate)
	}
	result := make([]Candidate, 0, len(ordered))
	for _, providerID := range order {
		group := groups[providerID]
		start := 0
		for start < len(group) {
			end := start + 1
			for end < len(group) && group[end].Priority == group[start].Priority {
				end++
			}
			r.shuffleWeighted(group[start:end])
			start = end
		}
		result = append(result, group...)
	}
	return result
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
