package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/alltomatos/dontstop/gateway/internal/usage"
)

// AccountRouter selects an account and applies failover.
type AccountRouter interface {
	Route(ctx context.Context, tenantID, model string, attempt routing.Attempt) (routing.Candidate, error)
}

// UsageRecorder persists request usage and cost.
type UsageRecorder interface {
	Record(ctx context.Context, event usage.Event) error
}

// IdempotencyStore stores non-streaming responses by idempotency key.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (provider.ChatResponse, bool, error)
	Set(ctx context.Context, key string, response provider.ChatResponse) error
}

// Service executes chat requests through the account router and adapters.
type Service struct {
	router        AccountRouter
	decrypter     routing.Decrypter
	adapters      func(kind string) (provider.Adapter, error)
	store         IdempotencyStore
	cache         cache.ExactCache
	promptVersion string
	recorder      UsageRecorder
}

// NewService creates a chat service.
func NewService(router AccountRouter, decrypter routing.Decrypter, adapters func(string) (provider.Adapter, error), store IdempotencyStore, exactCache cache.ExactCache, promptVersion string, recorder UsageRecorder) *Service {
	return &Service{
		router:        router,
		decrypter:     decrypter,
		adapters:      adapters,
		store:         store,
		cache:         exactCache,
		promptVersion: promptVersion,
		recorder:      recorder,
	}
}

// Chat executes a non-streaming chat completion.
func (s *Service) Chat(ctx context.Context, tenantID string, request provider.ChatRequest, idempotencyKey string) (provider.ChatResponse, string, error) {
	started := time.Now()
	if idempotencyKey != "" && s.store != nil {
		if cached, found, err := s.store.Get(ctx, tenantID+":"+idempotencyKey); err == nil && found {
			return cached, "bypass", nil
		}
	}

	cacheKey := ""
	cacheStatus := "bypass"
	if s.cache != nil {
		key, err := cache.BuildKey(cache.KeyInput{
			TenantID: tenantID, Scope: "chat:write", Model: request.Model,
			PromptVersion: s.promptVersion, Messages: request.Messages,
			Temperature: request.Temperature, MaxTokens: request.MaxTokens,
		})
		if err == nil {
			cacheKey = key
			entry, found, err := s.cache.Get(ctx, key)
			if err == nil && found && entry.ErrorCode == "" {
				s.recordUsage(ctx, tenantID, "", "", request.Model, "success", "hit", entry.Response.Usage, time.Since(started))
				return entry.Response, "hit", nil
			}
			if err == nil {
				cacheStatus = "miss"
			}
		}
	}

	var response provider.ChatResponse
	selected, err := s.router.Route(ctx, tenantID, request.Model, func(ctx context.Context, candidate routing.Candidate) (routing.AttemptResult, error) {
		adapter, account, err := s.resolve(candidate)
		if err != nil {
			return routing.AttemptResult{Failure: routing.FailureUnknown}, err
		}
		result, err := adapter.Chat(ctx, account, request)
		if err != nil {
			return routing.AttemptResult{Failure: routing.ClassifyProviderError(err)}, err
		}
		response = result
		return routing.AttemptResult{}, nil
	})
	if err != nil {
		return provider.ChatResponse{}, "", err
	}
	if idempotencyKey != "" && s.store != nil {
		_ = s.store.Set(ctx, tenantID+":"+idempotencyKey, response)
	}
	if cacheKey != "" {
		_ = s.cache.Set(ctx, cacheKey, cache.Meta{
			TenantID: tenantID, Model: request.Model, Provider: selected.ProviderName,
			PromptVersion: s.promptVersion,
		}, cache.Entry{Response: response}, time.Hour)
	}
	s.recordUsage(ctx, tenantID, selected.ProviderID, selected.AccountID, request.Model, "success", cacheStatus, response.Usage, time.Since(started))
	return response, cacheStatus, nil
}

// Stream executes a streaming chat completion and writes OpenAI-compatible SSE.
func (s *Service) Stream(ctx context.Context, tenantID string, request provider.ChatRequest, writer http.ResponseWriter) error {
	started := time.Now()
	state := &streamState{writer: writer, id: newCompletionID(), model: request.Model, created: time.Now().Unix()}
	selected, err := s.router.Route(ctx, tenantID, request.Model, func(ctx context.Context, candidate routing.Candidate) (routing.AttemptResult, error) {
		adapter, account, err := s.resolve(candidate)
		if err != nil {
			return routing.AttemptResult{Failure: routing.FailureUnknown}, err
		}
		stream, err := adapter.ChatStream(ctx, account, request)
		if err != nil {
			return routing.AttemptResult{Failure: routing.ClassifyProviderError(err)}, err
		}
		defer func() { _ = stream.Close() }()

		for {
			chunk, err := stream.Recv(ctx)
			if err == io.EOF {
				if state.started {
					state.done()
				}
				return routing.AttemptResult{}, nil
			}
			if err != nil {
				if state.started {
					state.errorEvent()
					state.done()
					return routing.AttemptResult{FirstTokenSent: true, Failure: routing.ClassifyProviderError(err)}, err
				}
				return routing.AttemptResult{Failure: routing.ClassifyProviderError(err)}, err
			}
			if err := state.chunk(chunk); err != nil {
				return routing.AttemptResult{FirstTokenSent: state.started, Failure: routing.FailureUnknown}, err
			}
		}
	})
	if err == nil {
		s.recordUsage(ctx, tenantID, selected.ProviderID, selected.AccountID, request.Model, "success", "bypass", state.usage, time.Since(started))
	}
	return err
}

func (s *Service) resolve(candidate routing.Candidate) (provider.Adapter, provider.Account, error) {
	account, err := routing.ResolveAccount(candidate, s.decrypter)
	if err != nil {
		return nil, provider.Account{}, err
	}
	adapter, err := s.adapters(candidate.ProviderKind)
	if err != nil {
		return nil, provider.Account{}, err
	}
	return adapter, account, nil
}

type streamState struct {
	writer  http.ResponseWriter
	id      string
	model   string
	created int64
	started bool
	usage   provider.Usage
}

func (s *streamState) ensureHeaders() error {
	if s.started {
		return nil
	}
	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	s.writer.Header().Set("Connection", "keep-alive")
	s.writer.Header().Set("X-Accel-Buffering", "no")
	s.writer.Header().Set("X-Donstop-Cache", "bypass")
	s.writer.WriteHeader(http.StatusOK)
	s.started = true
	return nil
}

func (s *streamState) chunk(chunk provider.StreamChunk) error {
	if err := s.ensureHeaders(); err != nil {
		return err
	}
	payload := map[string]any{
		"id":      s.id,
		"object":  "chat.completion.chunk",
		"created": s.created,
		"model":   s.model,
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]string{"content": chunk.Content},
			"finish_reason": nil,
		}},
	}
	if chunk.Content == "" {
		payload["choices"].([]map[string]any)[0]["delta"] = map[string]string{}
	}
	if chunk.FinishReason != "" {
		payload["choices"].([]map[string]any)[0]["finish_reason"] = chunk.FinishReason
	}
	if chunk.Usage != nil {
		s.usage = *chunk.Usage
		payload["usage"] = map[string]int{
			"prompt_tokens":     chunk.Usage.PromptTokens,
			"completion_tokens": chunk.Usage.CompletionTokens,
			"total_tokens":      chunk.Usage.TotalTokens,
		}
	}
	return writeEvent(s.writer, payload)
}

func (s *streamState) errorEvent() {
	_ = writeEvent(s.writer, map[string]any{
		"error": map[string]string{
			"type":    "api_error",
			"message": "upstream stream failed",
		},
	})
}

func (s *streamState) done() {
	_, _ = fmt.Fprint(s.writer, "data: [DONE]\n\n")
	if flusher, ok := s.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeEvent(writer http.ResponseWriter, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", data); err != nil {
		return err
	}
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func (s *Service) recordUsage(ctx context.Context, tenantID, providerID, accountID, model, status, cacheStatus string, tokens provider.Usage, latency time.Duration) {
	if s.recorder == nil {
		return
	}
	_ = s.recorder.Record(ctx, usage.Event{
		RequestID: newRequestID(), TenantID: tenantID,
		ProviderID: providerID, AccountID: accountID, Model: model,
		Status: status, CacheStatus: cacheStatus,
		TokensIn: tokens.PromptTokens, TokensOut: tokens.CompletionTokens,
		LatencyMS: int(latency.Milliseconds()), CreatedAt: time.Now().UTC(),
	})
}

func newRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "req_fallback"
	}
	return "req_" + hex.EncodeToString(buffer)
}

func newCompletionID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "chatcmpl_ds_fallback"
	}
	return "chatcmpl_ds_" + hex.EncodeToString(buffer)
}
