package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
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

const (
	// chatResponseTimeout bounds non-streaming completions. http.Client.Timeout
	// must not be used for providers: it also bounds reading a streamed SSE body,
	// which silently truncated long generations.
	chatResponseTimeout     = 3 * time.Minute
	providerDialTimeout     = 10 * time.Second
	providerTLSHandshake    = 10 * time.Second
	providerHeaderTimeout   = 60 * time.Second
	providerIdleConnTimeout = 90 * time.Second
	// providerStreamIdleTimeout aborts a stream that stops yielding bytes.
	providerStreamIdleTimeout = 2 * time.Minute
)

// ChatMessage is a provider-neutral chat message. Content holds the plain text
// form and Parts holds the multimodal form when the caller sent an array. Both
// are mutually derived: Content is always the joined text of Parts.
type ChatMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	Parts      []ContentPart `json:"-"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	Name       string        `json:"name,omitempty"`
}

// UnmarshalJSON accepts the OpenAI content union: a string, null or an array of
// content parts.
func (m *ChatMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		ToolCalls  []ToolCall      `json:"tool_calls"`
		ToolCallID string          `json:"tool_call_id"`
		Name       string          `json:"name"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	m.Content = ""
	m.Parts = nil
	m.ToolCalls = raw.ToolCalls
	m.ToolCallID = raw.ToolCallID
	m.Name = raw.Name
	if len(raw.Content) == 0 || string(raw.Content) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(raw.Content, &text); err == nil {
		m.Content = text
		return nil
	}
	var parts []ContentPart
	if err := json.Unmarshal(raw.Content, &parts); err != nil {
		return fmt.Errorf("content must be a string or an array of content parts")
	}
	for index := range parts {
		if err := parts[index].Validate(); err != nil {
			return fmt.Errorf("content[%d]: %w", index, err)
		}
	}
	m.Parts = parts
	m.Content = joinTextParts(parts)
	return nil
}

// MarshalJSON keeps the scalar form for plain text messages and emits the
// OpenAI content array when the caller sent content parts.
func (m ChatMessage) MarshalJSON() ([]byte, error) {
	type wire struct {
		Role       string     `json:"role"`
		Content    any        `json:"content"`
		ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
		Name       string     `json:"name,omitempty"`
	}
	var content any = m.Content
	if m.Parts != nil {
		content = m.Parts
	}
	return json.Marshal(wire{
		Role: m.Role, Content: content, ToolCalls: m.ToolCalls, ToolCallID: m.ToolCallID, Name: m.Name,
	})
}

// ChatRequest is a provider-neutral chat request for the P0-α slice.
type ChatRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  json.RawMessage `json:"tool_choice,omitempty"`
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
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
	Provider     string
	MemberModel  string
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
	return &http.Client{Transport: providerTransport()}
}

// providerTransport applies per-phase deadlines so a stalled connection fails
// fast without capping the total lifetime of a streamed response.
func providerTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: providerDialTimeout, KeepAlive: 30 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = providerTLSHandshake
	transport.ResponseHeaderTimeout = providerHeaderTimeout
	transport.IdleConnTimeout = providerIdleConnTimeout
	return transport
}

func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, target any, proxyEnabled bool) error {
	ctx, cancel := context.WithTimeout(ctx, chatResponseTimeout)
	defer cancel()
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
	return newIdleTimeoutReadCloser(response.Body, providerStreamIdleTimeout), nil
}

// idleTimeoutReadCloser closes a streamed response body once it stops yielding
// bytes for idle. It replaces http.Client.Timeout for SSE, which would otherwise
// abort the whole generation at a fixed wall-clock deadline.
type idleTimeoutReadCloser struct {
	body   io.ReadCloser
	idle   time.Duration
	timer  *time.Timer
	mu     sync.Mutex
	err    error
	closed bool
}

func newIdleTimeoutReadCloser(body io.ReadCloser, idle time.Duration) io.ReadCloser {
	if idle <= 0 {
		return body
	}
	reader := &idleTimeoutReadCloser{body: body, idle: idle}
	reader.timer = time.AfterFunc(idle, reader.expire)
	return reader
}

func (r *idleTimeoutReadCloser) expire() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.err = fmt.Errorf("%w: upstream stream idle for %s", ErrProviderUnavailable, r.idle)
	r.mu.Unlock()
	_ = r.body.Close()
}

func (r *idleTimeoutReadCloser) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	r.mu.Lock()
	if n > 0 && !r.closed && r.err == nil {
		r.timer.Reset(r.idle)
	}
	expired := r.err
	r.mu.Unlock()
	if err != nil && expired != nil {
		return n, expired
	}
	return n, err
}

func (r *idleTimeoutReadCloser) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()
	r.timer.Stop()
	return r.body.Close()
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
