package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher publishes an outbox event with an idempotency message ID.
type Publisher interface {
	Publish(ctx context.Context, messageID string, payload []byte) error
}

// OutboxPublisher drains unpublished usage events.
type OutboxPublisher struct {
	pool      *pgxpool.Pool
	publisher Publisher
	batch     int
}

// NewOutboxPublisher creates an outbox publisher.
func NewOutboxPublisher(pool *pgxpool.Pool, publisher Publisher) *OutboxPublisher {
	return &OutboxPublisher{pool: pool, publisher: publisher, batch: 100}
}

// PublishOnce publishes one batch and marks each event as published.
func (p *OutboxPublisher) PublishOnce(ctx context.Context) (int, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	rows, err := tx.Query(ctx, `
		SELECT id::text, event_id::text, payload
		FROM usage_outbox
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, p.batch)
	if err != nil {
		return 0, fmt.Errorf("query outbox: %w", err)
	}
	type entry struct {
		id      string
		eventID string
		payload []byte
	}
	entries := make([]entry, 0)
	for rows.Next() {
		var item entry
		if err := rows.Scan(&item.id, &item.eventID, &item.payload); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan outbox: %w", err)
		}
		entries = append(entries, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate outbox: %w", err)
	}

	for _, item := range entries {
		if err := p.publisher.Publish(ctx, item.eventID, item.payload); err != nil {
			return 0, fmt.Errorf("publish outbox event: %w", err)
		}
		if _, err := tx.Exec(ctx, "UPDATE usage_outbox SET published_at = now(), attempts = attempts + 1 WHERE id = $1", item.id); err != nil {
			return 0, fmt.Errorf("mark outbox published: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox: %w", err)
	}
	return len(entries), nil
}

// Run drains the outbox until the context is cancelled.
func (p *OutboxPublisher) Run(ctx context.Context, interval time.Duration, onError func(error)) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := p.PublishOnce(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
