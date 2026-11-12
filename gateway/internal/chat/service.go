package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/proxy"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
	"github.com/alltomatos/dontstop/gateway/internal/usage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("dontstop/gateway/chat")

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
	proxies       ProxyResolver
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
	ctx, span := tracer.Start(ctx, "chat.completion")
	defer span.End()
	span.SetAttributes(attribute.String("gen_ai.request.model", request.Model))
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
			cacheCtx, cacheSpan := tracer.Start(ctx, "cache.exact.get")
			entry, found, err := s.cache.Get(cacheCtx, key)
			cacheSpan.SetAttributes(attribute.Bool("cache.hit", found))
			if err != nil {
				cacheSpan.RecordError(err)
				cacheSpan.SetStatus(codes.Error, "cache read failed")
			}
			cacheSpan.End()
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
	routeCtx, routeSpan := tracer.Start(ctx, "routing.select")
	selected, err := s.router.Route(routeCtx, tenantID, request.Model, func(ctx context.Context, candidate routing.Candidate) (routing.AttemptResult, error) {
		result, outcome, err := s.attemptChat(ctx, candidate, request)
		if err == nil {
			response = result
		}
		return outcome, err
	})
	routeSpan.End()
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
	span.SetAttributes(attribute.String("cache.status", cacheStatus))
	s.recordUsage(ctx, tenantID, selected.ProviderID, selected.AccountID, request.Model, "success", cacheStatus, response.Usage, time.Since(started))
	return response, cacheStatus, nil
}

// Stream executes a streaming chat completion and writes OpenAI-compatible SSE.
func (s *Service) Stream(ctx context.Context, tenantID string, request provider.ChatRequest, writer http.ResponseWriter) error {
	ctx, span := tracer.Start(ctx, "chat.stream")
	defer span.End()
	span.SetAttributes(attribute.String("gen_ai.request.model", request.Model))
	started := time.Now()
	state := &streamState{writer: writer, id: newCompletionID(), model: request.Model, created: time.Now().Unix()}
	selected, err := s.router.Route(ctx, tenantID, request.Model, func(ctx context.Context, candidate routing.Candidate) (routing.AttemptResult, error) {
		return s.attemptStream(ctx, candidate, request, state)
	})
	if err == nil {
		s.recordUsage(ctx, tenantID, selected.ProviderID, selected.AccountID, request.Model, "success", "bypass", state.usage, time.Since(started))
	}
	return err
}

func (s *Service) resolve(ctx context.Context, candidate routing.Candidate, forceDirect bool, forced *proxy.Lease) (provider.Adapter, provider.Account, proxy.Lease, error) {
	if s.proxies == nil {
		account, err := routing.ResolveAccount(candidate, s.decrypter)
		if err != nil {
			return nil, provider.Account{}, proxy.Lease{}, err
		}
		adapter, err := s.adapters(candidate.ProviderKind)
		return adapter, account, proxy.Lease{}, err
	}
	account, err := routing.ResolveCredentials(candidate, s.decrypter)
	if err != nil {
		return nil, provider.Account{}, proxy.Lease{}, err
	}
	adapter, err := s.adapters(candidate.ProviderKind)
	if err != nil {
		return nil, provider.Account{}, proxy.Lease{}, err
	}
	if forceDirect {
		return adapter, account, proxy.Lease{}, nil
	}
	lease := proxy.Lease{}
	if forced != nil {
		lease = *forced
	} else {
		lease, err = s.proxies.Resolve(ctx, candidate.TenantID, candidate.AccountID, candidate.ProxyMode)
		if err != nil {
			return nil, provider.Account{}, proxy.Lease{}, err
		}
	}
	account.ProxyURL = lease.ProxyURL
	return adapter, account, lease, nil
}

func (s *Service) attemptChat(ctx context.Context, candidate routing.Candidate, request provider.ChatRequest) (provider.ChatResponse, routing.AttemptResult, error) {
	var forced *proxy.Lease
	forceDirect := false
	var lastErr error
	for round := 0; round <= maxProxyRotations; round++ {
		adapter, account, lease, err := s.resolve(ctx, candidate, forceDirect, forced)
		forced = nil
		forceDirect = false
		if err != nil {
			return provider.ChatResponse{}, routing.AttemptResult{Failure: attemptFailureKind(err)}, err
		}
		providerCtx, providerSpan := tracer.Start(ctx, "provider.chat")
		providerSpan.SetAttributes(
			attribute.String("gen_ai.provider.name", candidate.ProviderName),
			attribute.String("gen_ai.request.model", request.Model),
		)
		result, err := adapter.Chat(providerCtx, account, request)
		if err != nil {
			providerSpan.RecordError(err)
			providerSpan.SetStatus(codes.Error, "provider request failed")
		}
		providerSpan.End()
		if err == nil {
			s.proxySuccess(ctx, candidate.TenantID, lease)
			return result, routing.AttemptResult{}, nil
		}
		lastErr = err
		if s.canRotate(lease, err, round) {
			next, rotateErr := s.proxies.Rotate(ctx, candidate.TenantID, candidate.AccountID, lease.EndpointID, candidate.ProxyMode)
			if errors.Is(rotateErr, provider.ErrProxyRequired) {
				return provider.ChatResponse{}, routing.AttemptResult{Failure: routing.FailureProxy}, rotateErr
			}
			if rotateErr == nil {
				forceDirect, forced = rotateTarget(next)
				continue
			}
		}
		return provider.ChatResponse{}, routing.AttemptResult{Failure: routing.ClassifyProviderError(err)}, err
	}
	return provider.ChatResponse{}, routing.AttemptResult{Failure: routing.ClassifyProviderError(lastErr)}, lastErr
}

func (s *Service) attemptStream(ctx context.Context, candidate routing.Candidate, request provider.ChatRequest, state *streamState) (routing.AttemptResult, error) {
	var forced *proxy.Lease
	forceDirect := false
	var lastErr error
	for round := 0; round <= maxProxyRotations; round++ {
		adapter, account, lease, err := s.resolve(ctx, candidate, forceDirect, forced)
		forced = nil
		forceDirect = false
		if err != nil {
			return routing.AttemptResult{Failure: attemptFailureKind(err)}, err
		}
		providerCtx, providerSpan := tracer.Start(ctx, "provider.stream")
		providerSpan.SetAttributes(attribute.String("gen_ai.provider.name", candidate.ProviderName))
		stream, err := adapter.ChatStream(providerCtx, account, request)
		if err == nil {
			err = s.consumeStream(ctx, stream, state)
		}
		if err != nil {
			providerSpan.RecordError(err)
			providerSpan.SetStatus(codes.Error, "provider stream failed")
		}
		providerSpan.End()
		if err == nil {
			s.proxySuccess(ctx, candidate.TenantID, lease)
			return routing.AttemptResult{}, nil
		}
		lastErr = err
		if state.started {
			return routing.AttemptResult{FirstTokenSent: true, Failure: routing.ClassifyProviderError(err)}, err
		}
		if s.canRotate(lease, err, round) {
			next, rotateErr := s.proxies.Rotate(ctx, candidate.TenantID, candidate.AccountID, lease.EndpointID, candidate.ProxyMode)
			if errors.Is(rotateErr, provider.ErrProxyRequired) {
				return routing.AttemptResult{Failure: routing.FailureProxy}, rotateErr
			}
			if rotateErr == nil {
				forceDirect, forced = rotateTarget(next)
				continue
			}
		}
		return routing.AttemptResult{Failure: routing.ClassifyProviderError(err)}, err
	}
	return routing.AttemptResult{Failure: routing.ClassifyProviderError(lastErr)}, lastErr
}

func (s *Service) consumeStream(ctx context.Context, stream provider.Stream, state *streamState) error {
	defer func() { _ = stream.Close() }()
	for {
		chunk, err := stream.Recv(ctx)
		if err == io.EOF {
			if state.started {
				state.done()
			}
			return nil
		}
		if err != nil {
			if state.started {
				state.errorEvent()
				state.done()
			}
			return err
		}
		if err := state.chunk(chunk); err != nil {
			return err
		}
	}
}

func (s *Service) canRotate(lease proxy.Lease, err error, round int) bool {
	return s.proxies != nil && lease.EndpointID != "" && round < maxProxyRotations && errors.Is(err, provider.ErrProxyFailure)
}

func (s *Service) proxySuccess(ctx context.Context, tenantID string, lease proxy.Lease) {
	if s.proxies != nil {
		s.proxies.Success(ctx, tenantID, lease)
	}
}

func rotateTarget(next proxy.Lease) (bool, *proxy.Lease) {
	if next.EndpointID == "" {
		return true, nil
	}
	return false, &next
}

func attemptFailureKind(err error) routing.FailureKind {
	if errors.Is(err, provider.ErrProxyRequired) {
		return routing.FailureProxy
	}
	return routing.FailureUnknown
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
	ctx, span := tracer.Start(ctx, "usage.record")
	defer span.End()
	span.SetAttributes(
		attribute.String("gen_ai.request.model", model),
		attribute.Int("gen_ai.usage.input_tokens", tokens.PromptTokens),
		attribute.Int("gen_ai.usage.output_tokens", tokens.CompletionTokens),
	)
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
