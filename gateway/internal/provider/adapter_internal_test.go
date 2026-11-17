package provider

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestLooksLikeBan(t *testing.T) {
	banSignals := []string{
		`{"error":"your IP address is banned"}`,
		"Access blocked from this region",
		"suspicious activity detected",
	}
	for _, body := range banSignals {
		if !looksLikeBan([]byte(body)) {
			t.Fatalf("expected a ban signal in %q", body)
		}
	}
	notBans := []string{
		`{"error":"invalid api key"}`,
		"rate limit exceeded",
		`{"error":"model not found"}`,
	}
	for _, body := range notBans {
		if looksLikeBan([]byte(body)) {
			t.Fatalf("unexpected ban signal in %q", body)
		}
	}
}

func TestProviderClientHasNoWallClockTimeout(t *testing.T) {
	// http.Client.Timeout also bounds reading a streamed body, which truncated
	// long generations after 60s. Streaming must rely on the idle watchdog.
	if got := defaultClient().Timeout; got != 0 {
		t.Fatalf("expected no client timeout for streaming, got %s", got)
	}
}

func TestIdleTimeoutReadCloserAbortsStalledStream(t *testing.T) {
	body := newBlockingBody()
	reader := newIdleTimeoutReadCloser(body, 20*time.Millisecond)
	defer reader.Close()

	start := time.Now()
	_, err := reader.Read(make([]byte, 8))
	if err == nil {
		t.Fatal("expected the idle watchdog to abort a stalled stream")
	}
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("idle watchdog took too long: %s", elapsed)
	}
}

func TestIdleTimeoutReadCloserDeliversData(t *testing.T) {
	body := newBlockingBody()
	reader := newIdleTimeoutReadCloser(body, time.Second)
	defer reader.Close()

	body.chunks <- "data: hello\n\n"
	buffer := make([]byte, 32)
	n, err := reader.Read(buffer)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buffer[:n]) != "data: hello\n\n" {
		t.Fatalf("unexpected payload %q", buffer[:n])
	}

	body.chunks <- "data: world\n\n"
	n, err = reader.Read(buffer)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buffer[:n]) != "data: world\n\n" {
		t.Fatalf("unexpected payload %q", buffer[:n])
	}
}

type blockingBody struct {
	chunks chan string
	closed chan struct{}
	once   sync.Once
}

func newBlockingBody() *blockingBody {
	return &blockingBody{chunks: make(chan string, 1), closed: make(chan struct{})}
}

func (b *blockingBody) Read(p []byte) (int, error) {
	select {
	case chunk, ok := <-b.chunks:
		if !ok {
			return 0, io.EOF
		}
		return copy(p, chunk), nil
	case <-b.closed:
		return 0, errors.New("body closed")
	}
}

func (b *blockingBody) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}
