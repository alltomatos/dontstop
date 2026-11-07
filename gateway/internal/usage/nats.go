package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const usageStream = "USAGE"

// NATSPublisher publishes usage events to JetStream with idempotency IDs.
type NATSPublisher struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewNATSPublisher connects to NATS and ensures the USAGE stream exists.
func NewNATSPublisher(url string) (*NATSPublisher, error) {
	if url == "" {
		return nil, fmt.Errorf("NATS URL is empty")
	}
	conn, err := nats.Connect(url, nats.Timeout(5*time.Second), nats.MaxReconnects(-1))
	if err != nil {
		return nil, fmt.Errorf("connect NATS: %w", err)
	}
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}
	if _, err := js.StreamInfo(usageStream); err != nil {
		if _, addErr := js.AddStream(&nats.StreamConfig{
			Name: usageStream, Subjects: []string{"usage.>"}, Storage: nats.FileStorage, Duplicates: 2 * time.Minute,
		}); addErr != nil {
			conn.Close()
			return nil, fmt.Errorf("ensure usage stream: %w", addErr)
		}
	}
	return &NATSPublisher{conn: conn, js: js}, nil
}

// Publish sends one usage event with its outbox event ID as Msg-Id.
func (p *NATSPublisher) Publish(ctx context.Context, messageID string, payload []byte) error {
	if _, err := p.js.Publish("usage.events", payload, nats.MsgId(messageID), nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish usage event: %w", err)
	}
	return nil
}

// Close drains and closes the NATS connection.
func (p *NATSPublisher) Close() {
	_ = p.conn.Drain()
	p.conn.Close()
}
