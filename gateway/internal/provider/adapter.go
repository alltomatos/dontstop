package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrProviderAuth        = errors.New("provider authentication failed")
	ErrProviderRateLimited = errors.New("provider rate limited")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrProviderResponse    = errors.New("invalid provider response")
)

const maxProviderResponseBytes = 4 << 20

// ChatMessage is a provider-neutral chat message.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is a provider-neutral chat request for the P0-α slice.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// Usage is provider-neutral token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is a provider-neutral non-streaming chat response.
type ChatResponse struct {
	ID           string
	Model        string
	Content      string
	FinishReason string
	Usage        Usage
}

// Account is the runtime credential/endpoint selected for a provider.
type Account struct {
	ID      string
	BaseURL string
	APIKey  string
}

// Adapter translates the gateway contract to one provider protocol.
type Adapter interface {
	Chat(ctx context.Context, account Account, request ChatRequest) (ChatResponse, error)
	ChatStream(ctx context.Context, account Account, request ChatRequest) (Stream, error)
}

// NewAdapter creates an adapter for a supported provider kind.
func NewAdapter(kind string) (Adapter, error) {
	switch strings.ToLower(kind) {
	case "openai":
		return &openAIAdapter{client: defaultClient()}, nil
	case "anthropic":
		return &anthropicAdapter{client: defaultClient()}, nil
	default:
		return nil, fmt.Errorf("unsupported provider kind %q", kind)
	}
}

func defaultClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, target any) error {
	response, err := doProviderRequest(ctx, client, url, headers, payload, "application/json")
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if err := json.NewDecoder(io.LimitReader(response.Body, maxProviderResponseBytes)).Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrProviderResponse, err)
	}
	return nil
}

func postSSE(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any) (io.ReadCloser, error) {
	response, err := doProviderRequest(ctx, client, url, headers, payload, "text/event-stream")
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func doProviderRequest(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, accept string) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode provider request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create provider request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", accept)
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxProviderResponseBytes))
		_ = response.Body.Close()
		switch {
		case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
			return nil, ErrProviderAuth
		case response.StatusCode == http.StatusTooManyRequests:
			return nil, ErrProviderRateLimited
		case response.StatusCode >= 500:
			return nil, fmt.Errorf("%w: status %d", ErrProviderUnavailable, response.StatusCode)
		default:
			return nil, fmt.Errorf("%w: status %d", ErrProviderResponse, response.StatusCode)
		}
	}
	return response, nil
}
