package admin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CacheStats is the aggregate cache snapshot.
type CacheStats struct {
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	Requests int64   `json:"requests"`
	HitRate  float64 `json:"hit_rate"`
	AvgMS    float64 `json:"avg_latency_ms"`
}

// CacheEvent is a payload-free cache event.
type CacheEvent struct {
	Time      string `json:"time"`
	Status    string `json:"status"`
	Model     string `json:"model"`
	LatencyMS int    `json:"latency_ms"`
	Tokens    int64  `json:"tokens"`
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
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE cache_status = 'hit')::bigint,
		       count(*) FILTER (WHERE cache_status = 'miss')::bigint,
		       count(*)::bigint,
		       COALESCE(avg(latency_ms) FILTER (WHERE cache_status IS NOT NULL), 0)::float8
		FROM requests
		WHERE tenant_id = $1 AND created_at >= now() - ($2::int * interval '1 day')
	`, tenantID, days).Scan(&stats.Hits, &stats.Misses, &stats.Requests, &stats.AvgMS); err != nil {
		return CacheStats{}, fmt.Errorf("query cache stats: %w", err)
	}
	if stats.Requests > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Requests)
	}
	return stats, nil
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
