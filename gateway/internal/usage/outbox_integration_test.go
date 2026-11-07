package usage_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/usage"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

type fakePublisher struct {
	fail bool
	ids  []string
}

func (p *fakePublisher) Publish(_ context.Context, messageID string, _ []byte) error {
	if p.fail {
		return errors.New("NATS unavailable")
	}
	p.ids = append(p.ids, messageID)
	return nil
}

func TestOutboxPublisherRetriesUntilPublished(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO usage_outbox (event_id,payload) VALUES (gen_random_uuid(),'{}'),(gen_random_uuid(),'{}')"); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}

	publisher := &fakePublisher{fail: true}
	outbox := usage.NewOutboxPublisher(pool, publisher)
	if _, err := outbox.PublishOnce(ctx); err == nil {
		t.Fatal("expected publish failure")
	}
	var unpublished int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM usage_outbox WHERE published_at IS NULL").Scan(&unpublished); err != nil {
		t.Fatalf("count: %v", err)
	}
	if unpublished != 2 {
		t.Fatalf("expected 2 unpublished, got %d", unpublished)
	}

	publisher.fail = false
	published, err := outbox.PublishOnce(ctx)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published != 2 || len(publisher.ids) != 2 {
		t.Fatalf("unexpected publish count: %d ids=%d", published, len(publisher.ids))
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM usage_outbox WHERE published_at IS NULL").Scan(&unpublished); err != nil {
		t.Fatalf("count: %v", err)
	}
	if unpublished != 0 {
		t.Fatalf("expected 0 unpublished, got %d", unpublished)
	}
}
