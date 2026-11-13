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
	ErrProviderAuth           = errors.New("provider authentication failed")
	ErrProviderRateLimited    = errors.New("provider rate limited")
	ErrProviderQuota          = errors.New("provider quota exhausted")
	ErrProviderUnavailable    = errors.New("provider unavailable")
	ErrProviderResponse       = errors.New("invalid provider response")
	ErrProviderInvalidRequest = errors.New("invalid provider request")
	ErrProxyFailure           = errors.New("proxy transport failure")
	ErrProxyAuth              = errors.New("proxy authentication failed")
	ErrProxyBanned            = errors.New("upstream banned the proxy ip")
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
	ID       string
	BaseURL  string
	APIKey   string
	ProxyURL string
}

// Adapter translates the gateway contract to one provider protocol.
type Adapter interface {
	Chat(ctx context.Context, account Account, request ChatRequest) (ChatResponse, error)
	ChatStream(ctx context.Context, account Account, request ChatRequest) (Stream, error)
}

// NewAdapter creates an adapter for a supported provider kind.
func NewAdapter(kind string) (Adapter, error) {
	return NewAdapterWithTranslator(kind, NewHTTPTranslator(nil))
}

// NewAdapterWithTranslator creates an adapter with an injected translator.
// The translator is only used by providers that require prompt localization.
func NewAdapterWithTranslator(kind string, translator Translator) (Adapter, error) {
	switch strings.ToLower(kind) {
	case "openai":
		return &openAIAdapter{client: defaultClient()}, nil
	case "anthropic":
		return &anthropicAdapter{client: defaultClient()}, nil
	case "agentrouter":
		return newAgentRouterAdapter(defaultClient(), translator), nil
	default:
		return nil, fmt.Errorf("unsupported provider kind %q", kind)
	}
}

func defaultClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, target any, proxyEnabled bool) error {
	response, err := doProviderRequest(ctx, client, url, headers, payload, "application/json", proxyEnabled)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if err := json.NewDecoder(io.LimitReader(response.Body, maxProviderResponseBytes)).Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrProviderResponse, err)
	}
	return nil
}

func postSSE(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, proxyEnabled bool) (io.ReadCloser, error) {
	response, err := doProviderRequest(ctx, client, url, headers, payload, "text/event-stream", proxyEnabled)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func doProviderRequest(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, accept string, proxyEnabled bool) (*http.Response, error) {
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
	if strings.Contains(url, "opencode.ai/") {
		request.Header.Set("x-opencode-session", newOpenCodeSessionID())
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := client.Do(request)
	if err != nil {
		if proxyEnabled {
			return nil, fmt.Errorf("%w: %v", ErrProxyFailure, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		_ = response.Body.Close()
		switch {
		case response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusRequestEntityTooLarge || response.StatusCode == http.StatusUnprocessableEntity:
			return nil, fmt.Errorf("%w: status %d", ErrProviderInvalidRequest, response.StatusCode)
		case response.StatusCode == http.StatusProxyAuthRequired && proxyEnabled:
			return nil, fmt.Errorf("%w: status %d", ErrProxyAuth, response.StatusCode)
		case response.StatusCode == http.StatusUnauthorized:
			return nil, ErrProviderAuth
		case response.StatusCode == http.StatusForbidden:
			if proxyEnabled && looksLikeBan(snippet) {
				return nil, fmt.Errorf("%w: status %d", ErrProxyBanned, response.StatusCode)
			}
			return nil, ErrProviderAuth
		case response.StatusCode == http.StatusPaymentRequired:
			return nil, ErrProviderQuota
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

// looksLikeBan reports whether a 403 body carries an IP/region ban signal.
func looksLikeBan(body []byte) bool {
	text := strings.ToLower(string(body))
	for _, signal := range []string{"banned", "blocked", "not allowed", "suspicious", "abuse", "suspended", "ip address"} {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}
