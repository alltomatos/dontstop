package usage_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/usage"
	"github.com/nats-io/nats.go"
)

func TestNATSPublisherPublishesWithDeduplication(t *testing.T) {
	url := os.Getenv("TEST_NATS_URL")
	if url == "" {
		t.Skip("TEST_NATS_URL is not set")
	}
	cleanup, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect cleanup: %v", err)
	}
	if cleanupJS, jsErr := cleanup.JetStream(); jsErr == nil {
		_ = cleanupJS.DeleteStream("USAGE")
	}
	cleanup.Close()

	publisher, err := usage.NewNATSPublisher(url)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	defer publisher.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publisher.Publish(ctx, "event-1", []byte(`{"request_id":"r1"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := publisher.Publish(ctx, "event-1", []byte(`{"request_id":"r1"}`)); err != nil {
		t.Fatalf("duplicate publish: %v", err)
	}

	conn, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	js, err := conn.JetStream()
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	info, err := js.StreamInfo("USAGE")
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	t.Logf("duplicates=%v msgs=%d", info.Config.Duplicates, info.State.Msgs)
	if info.State.Msgs != 1 {
		t.Fatalf("expected 1 deduplicated message, got %d", info.State.Msgs)
	}
}
