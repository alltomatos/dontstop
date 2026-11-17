package admin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CacheStats is the aggregate cache snapshot.
//
// HitRate is computed over cacheable requests only (hits + misses). Requests
// that never consult the cache — streaming and combo traffic, recorded as
// "bypass" — are reported separately so they do not dilute the hit rate.
type CacheStats struct {
	Days           int     `json:"days"`
	Hits           int64   `json:"hits"`
	Misses         int64   `json:"misses"`
	Bypass         int64   `json:"bypass"`
	Cacheable      int64   `json:"cacheable"`
	Requests       int64   `json:"requests"`
	HitRate        float64 `json:"hit_rate"`
	HitLatencyMS   float64 `json:"hit_latency_ms"`
	MissLatencyMS  float64 `json:"miss_latency_ms"`
	LatencySavedMS float64 `json:"latency_saved_ms"`
	TokensReused   int64   `json:"tokens_reused"`
}

// CacheDay is one day of the hit-rate series.
type CacheDay struct {
	Day     string  `json:"day"`
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	Bypass  int64   `json:"bypass"`
	HitRate float64 `json:"hit_rate"`
}

// CacheEvent is a payload-free cache event.
type CacheEvent struct {
	Time      string `json:"time"`
	Status    string `json:"status"`
	Model     string `json:"model"`
	LatencyMS int    `json:"latency_ms"`
	Tokens    int64  `json:"tokens"`
}

// CacheReport is the console cache payload.
type CacheReport struct {
	Stats  CacheStats   `json:"stats"`
	ByDay  []CacheDay   `json:"by_day"`
	Events []CacheEvent `json:"events"`
}

// CacheService queries cache aggregates.
type CacheService struct {
	pool *pgxpool.Pool
}

// NewCacheService creates a cache service.
func NewCacheService(pool *pgxpool.Pool) *CacheService {
	return &CacheService{pool: pool}
}

// Stats returns cache hit/miss aggregates for the last days.
func (s *CacheService) Stats(ctx context.Context, tenantID string, days int) (CacheStats, error) {
	if days <= 0 || days > 365 {
		days = 1
	}
	var stats CacheStats
	stats.Days = days
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE cache_status = 'hit')::bigint,
		       count(*) FILTER (WHERE cache_status = 'miss')::bigint,
		       count(*) FILTER (WHERE cache_status = 'bypass')::bigint,
		       count(*)::bigint,
		       COALESCE(avg(latency_ms) FILTER (WHERE cache_status = 'hit'), 0)::float8,
		       COALESCE(avg(latency_ms) FILTER (WHERE cache_status = 'miss'), 0)::float8,
		       COALESCE(sum(tokens_in + tokens_out) FILTER (WHERE cache_status = 'hit'), 0)::bigint
		FROM requests
		WHERE tenant_id = $1 AND created_at >= current_date - ($2::int - 1) * interval '1 day'
	`, tenantID, days).Scan(
		&stats.Hits, &stats.Misses, &stats.Bypass, &stats.Requests,
		&stats.HitLatencyMS, &stats.MissLatencyMS, &stats.TokensReused,
	); err != nil {
		return CacheStats{}, fmt.Errorf("query cache stats: %w", err)
	}
	stats.Cacheable = stats.Hits + stats.Misses
	if stats.Cacheable > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Cacheable)
	}
	if stats.HitLatencyMS > 0 && stats.MissLatencyMS > stats.HitLatencyMS {
		stats.LatencySavedMS = stats.MissLatencyMS - stats.HitLatencyMS
	}
	return stats, nil
}

// ByDay returns the cache hit-rate series.
func (s *CacheService) ByDay(ctx context.Context, tenantID string, days int) ([]CacheDay, error) {
	if days <= 0 || days > 365 {
		days = 1
	}
	rows, err := s.pool.Query(ctx, `
		SELECT created_at::date::text,
		       count(*) FILTER (WHERE cache_status = 'hit')::bigint,
		       count(*) FILTER (WHERE cache_status = 'miss')::bigint,
		       count(*) FILTER (WHERE cache_status = 'bypass')::bigint
		FROM requests
		WHERE tenant_id = $1 AND created_at >= current_date - ($2::int - 1) * interval '1 day'
		GROUP BY 1
		ORDER BY 1
	`, tenantID, days)
	if err != nil {
		return nil, fmt.Errorf("query cache by day: %w", err)
	}
	defer rows.Close()
	series := make([]CacheDay, 0)
	for rows.Next() {
		var day CacheDay
		if err := rows.Scan(&day.Day, &day.Hits, &day.Misses, &day.Bypass); err != nil {
			return nil, fmt.Errorf("scan cache by day: %w", err)
		}
		if cacheable := day.Hits + day.Misses; cacheable > 0 {
			day.HitRate = float64(day.Hits) / float64(cacheable)
		}
		series = append(series, day)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cache by day: %w", err)
	}
	return series, nil
}

// Events returns recent cache events without payloads.
func (s *CacheService) Events(ctx context.Context, tenantID string, limit int) ([]CacheEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(created_at, 'HH24:MI'), cache_status, model, COALESCE(latency_ms, 0), tokens_in + tokens_out
		FROM requests
		WHERE tenant_id = $1 AND cache_status IS NOT NULL
		ORDER BY created_at DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("query cache events: %w", err)
	}
	defer rows.Close()
	events := make([]CacheEvent, 0)
	for rows.Next() {
		var event CacheEvent
		if err := rows.Scan(&event.Time, &event.Status, &event.Model, &event.LatencyMS, &event.Tokens); err != nil {
			return nil, fmt.Errorf("scan cache event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cache events: %w", err)
	}
	return events, nil
}
