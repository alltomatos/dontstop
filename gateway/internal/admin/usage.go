package admin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CostSummary is the aggregate cost/usage window.
//
// Two sources are combined so the console never mixes them silently:
//   - usage_daily holds billable upstream requests (cache hits are not recorded
//     there, because a hit never reaches a provider), so Requests/Tokens/Cost
//     describe real upstream work only.
//   - requests holds every finished call, including the ones served from cache,
//     which is where CacheHits comes from.
//
// TotalRequests is the honest denominator: Requests + CacheHits.
type CostSummary struct {
	Days                 int     `json:"days"`
	PeriodStart          string  `json:"period_start,omitempty"`
	PeriodEnd            string  `json:"period_end,omitempty"`
	DaysWithData         int     `json:"days_with_data"`
	CostUSD              float64 `json:"cost_usd"`
	Requests             int64   `json:"requests"`
	CacheHits            int64   `json:"cache_hits"`
	TotalRequests        int64   `json:"total_requests"`
	TokensIn             int64   `json:"tokens_in"`
	TokensOut            int64   `json:"tokens_out"`
	TokensTotal          int64   `json:"tokens_total"`
	Errors               int64   `json:"errors"`
	ErrorRate            float64 `json:"error_rate"`
	AvgCostPerRequestUSD float64 `json:"avg_cost_per_request_usd"`
}

// CostByProvider is cost grouped by provider.
type CostByProvider struct {
	ProviderID   string  `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	CostUSD      float64 `json:"cost_usd"`
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	Errors       int64   `json:"errors"`
}

// CostByDay is cost grouped by day.
type CostByDay struct {
	Day      string  `json:"day"`
	CostUSD  float64 `json:"cost_usd"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Errors   int64   `json:"errors"`
}

// CostByModel is cost grouped by model.
type CostByModel struct {
	Model    string  `json:"model"`
	CostUSD  float64 `json:"cost_usd"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
}

// CostReport is the console cost payload.
type CostReport struct {
	Summary    CostSummary      `json:"summary"`
	ByProvider []CostByProvider `json:"by_provider"`
	ByDay      []CostByDay      `json:"by_day"`
	ByModel    []CostByModel    `json:"by_model"`
}

// ObservabilitySummary is the console SLO/usage snapshot.
type ObservabilitySummary struct {
	Requests  int64   `json:"requests"`
	Errors    int64   `json:"errors"`
	ErrorRate float64 `json:"error_rate"`
	P50MS     float64 `json:"p50_ms"`
	P95MS     float64 `json:"p95_ms"`
	P99MS     float64 `json:"p99_ms"`
	CostUSD   float64 `json:"cost_usd"`
	TokensIn  int64   `json:"tokens_in"`
	TokensOut int64   `json:"tokens_out"`
}

// UsageService queries cost and observability aggregates.
type UsageService struct {
	pool *pgxpool.Pool
}

// NewUsageService creates a usage service.
func NewUsageService(pool *pgxpool.Pool) *UsageService {
	return &UsageService{pool: pool}
}

// CostReport returns cost aggregates for the last days.
func (s *UsageService) CostReport(ctx context.Context, tenantID string, days int) (CostReport, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	var report CostReport
	report.Summary.Days = days
	var periodStart, periodEnd *string
	var daysWithData int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(cost_usd), 0)::float8,
		       COALESCE(sum(requests), 0)::bigint,
		       COALESCE(sum(tokens_in), 0)::bigint,
		       COALESCE(sum(tokens_out), 0)::bigint,
		       COALESCE(sum(errors), 0)::bigint,
		       min(day)::text,
		       max(day)::text,
		       count(DISTINCT day)::bigint
		FROM usage_daily
		WHERE tenant_id = $1 AND day >= current_date - ($2::int - 1)
	`, tenantID, days).Scan(
		&report.Summary.CostUSD, &report.Summary.Requests,
		&report.Summary.TokensIn, &report.Summary.TokensOut,
		&report.Summary.Errors, &periodStart, &periodEnd, &daysWithData,
	); err != nil {
		return CostReport{}, fmt.Errorf("query cost summary: %w", err)
	}
	if periodStart != nil {
		report.Summary.PeriodStart = *periodStart
	}
	if periodEnd != nil {
		report.Summary.PeriodEnd = *periodEnd
	}
	report.Summary.DaysWithData = int(daysWithData)
	report.Summary.TokensTotal = report.Summary.TokensIn + report.Summary.TokensOut
	if report.Summary.Requests > 0 {
		report.Summary.ErrorRate = float64(report.Summary.Errors) / float64(report.Summary.Requests)
		report.Summary.AvgCostPerRequestUSD = report.Summary.CostUSD / float64(report.Summary.Requests)
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE cache_status = 'hit')::bigint
		FROM requests
		WHERE tenant_id = $1 AND created_at >= current_date - ($2::int - 1) * interval '1 day'
	`, tenantID, days).Scan(&report.Summary.CacheHits); err != nil {
		return CostReport{}, fmt.Errorf("query cached requests: %w", err)
	}
	report.Summary.TotalRequests = report.Summary.Requests + report.Summary.CacheHits

	providerRows, err := s.pool.Query(ctx, `
		SELECT u.provider_id::text, p.name,
		       COALESCE(sum(u.cost_usd), 0)::float8,
		       COALESCE(sum(u.requests), 0)::bigint,
		       COALESCE(sum(u.tokens_in + u.tokens_out), 0)::bigint,
		       COALESCE(sum(u.errors), 0)::bigint
		FROM usage_daily u
		JOIN providers p ON p.id = u.provider_id AND p.tenant_id = u.tenant_id
		WHERE u.tenant_id = $1 AND u.day >= current_date - ($2::int - 1)
		GROUP BY u.provider_id, p.name
		ORDER BY 3 DESC
	`, tenantID, days)
	if err != nil {
		return CostReport{}, fmt.Errorf("query cost by provider: %w", err)
	}
	report.ByProvider = make([]CostByProvider, 0)
	for providerRows.Next() {
		var row CostByProvider
		if err := providerRows.Scan(&row.ProviderID, &row.ProviderName, &row.CostUSD, &row.Requests, &row.Tokens, &row.Errors); err != nil {
			providerRows.Close()
			return CostReport{}, fmt.Errorf("scan cost by provider: %w", err)
		}
		report.ByProvider = append(report.ByProvider, row)
	}
	providerRows.Close()
	if err := providerRows.Err(); err != nil {
		return CostReport{}, fmt.Errorf("iterate cost by provider: %w", err)
	}

	dayRows, err := s.pool.Query(ctx, `
		SELECT day::text,
		       COALESCE(sum(cost_usd), 0)::float8,
		       COALESCE(sum(requests), 0)::bigint,
		       COALESCE(sum(tokens_in + tokens_out), 0)::bigint,
		       COALESCE(sum(errors), 0)::bigint
		FROM usage_daily
		WHERE tenant_id = $1 AND day >= current_date - ($2::int - 1)
		GROUP BY day
		ORDER BY day
	`, tenantID, days)
	if err != nil {
		return CostReport{}, fmt.Errorf("query cost by day: %w", err)
	}
	report.ByDay = make([]CostByDay, 0)
	for dayRows.Next() {
		var row CostByDay
		if err := dayRows.Scan(&row.Day, &row.CostUSD, &row.Requests, &row.Tokens, &row.Errors); err != nil {
			dayRows.Close()
			return CostReport{}, fmt.Errorf("scan cost by day: %w", err)
		}
		report.ByDay = append(report.ByDay, row)
	}
	dayRows.Close()
	if err := dayRows.Err(); err != nil {
		return CostReport{}, fmt.Errorf("iterate cost by day: %w", err)
	}

	modelRows, err := s.pool.Query(ctx, `
		SELECT model, COALESCE(sum(cost_usd), 0)::float8, COALESCE(sum(requests), 0)::bigint,
		       COALESCE(sum(tokens_in + tokens_out), 0)::bigint
		FROM usage_daily
		WHERE tenant_id = $1 AND day >= current_date - ($2::int - 1)
		GROUP BY model
		ORDER BY 2 DESC
		LIMIT 10
	`, tenantID, days)
	if err != nil {
		return CostReport{}, fmt.Errorf("query cost by model: %w", err)
	}
	report.ByModel = make([]CostByModel, 0)
	for modelRows.Next() {
		var row CostByModel
		if err := modelRows.Scan(&row.Model, &row.CostUSD, &row.Requests, &row.Tokens); err != nil {
			modelRows.Close()
			return CostReport{}, fmt.Errorf("scan cost by model: %w", err)
		}
		report.ByModel = append(report.ByModel, row)
	}
	modelRows.Close()
	if err := modelRows.Err(); err != nil {
		return CostReport{}, fmt.Errorf("iterate cost by model: %w", err)
	}
	return report, nil
}

// Observability returns request latency/error aggregates for the last days.
func (s *UsageService) Observability(ctx context.Context, tenantID string, days int) (ObservabilitySummary, error) {
	if days <= 0 || days > 365 {
		days = 1
	}
	var summary ObservabilitySummary
	var p50, p95, p99 float64
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*)::bigint,
		       count(*) FILTER (WHERE status <> 'success')::bigint,
		       COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY latency_ms), 0)::float8,
		       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms), 0)::float8,
		       COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY latency_ms), 0)::float8,
		       COALESCE(sum(cost_usd), 0)::float8,
		       COALESCE(sum(tokens_in), 0)::bigint,
		       COALESCE(sum(tokens_out), 0)::bigint
		FROM requests
		WHERE tenant_id = $1 AND created_at >= now() - ($2::int * interval '1 day')
	`, tenantID, days).Scan(&summary.Requests, &summary.Errors, &p50, &p95, &p99, &summary.CostUSD, &summary.TokensIn, &summary.TokensOut); err != nil {
		return ObservabilitySummary{}, fmt.Errorf("query observability summary: %w", err)
	}
	summary.P50MS = p50
	summary.P95MS = p95
	summary.P99MS = p99
	if summary.Requests > 0 {
		summary.ErrorRate = float64(summary.Errors) / float64(summary.Requests)
	}
	return summary, nil
}
