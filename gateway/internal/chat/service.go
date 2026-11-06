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

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

// AccountRouter selects an account and applies failover.
type AccountRouter interface {
	Route(ctx context.Context, tenantID, model string, attempt routing.Attempt) (routing.Candidate, error)
}

// IdempotencyStore stores non-streaming responses by idempotency key.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (provider.ChatResponse, bool, error)
	Set(ctx context.Context, key string, response provider.ChatResponse) error
}

// Service executes chat requests through the account router and adapters.
type Service struct {
	router    AccountRouter
	decrypter routing.Decrypter
	adapters  func(kind string) (provider.Adapter, error)
	store     IdempotencyStore
}

// NewService creates a chat service.
func NewService(router AccountRouter, decrypter routing.Decrypter, adapters func(string) (provider.Adapter, error), store IdempotencyStore) *Service {
	return &Service{router: router, decrypter: decrypter, adapters: adapters, store: store}
}

// Chat executes a non-streaming chat completion.
func (s *Service) Chat(ctx context.Context, tenantID string, request provider.ChatRequest, idempotencyKey string) (provider.ChatResponse, error) {
	if idempotencyKey != "" && s.store != nil {
		if cached, found, err := s.store.Get(ctx, tenantID+":"+idempotencyKey); err == nil && found {
			return cached, nil
		}
	}

	var response provider.ChatResponse
	_, err := s.router.Route(ctx, tenantID, request.Model, func(ctx context.Context, candidate routing.Candidate) (routing.AttemptResult, error) {
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
		return provider.ChatResponse{}, err
	}
	if idempotencyKey != "" && s.store != nil {
		_ = s.store.Set(ctx, tenantID+":"+idempotencyKey, response)
	}
	return response, nil
}

// Stream executes a streaming chat completion and writes OpenAI-compatible SSE.
func (s *Service) Stream(ctx context.Context, tenantID string, request provider.ChatRequest, writer http.ResponseWriter) error {
	state := &streamState{writer: writer, id: newCompletionID(), model: request.Model, created: time.Now().Unix()}
	_, err := s.router.Route(ctx, tenantID, request.Model, func(ctx context.Context, candidate routing.Candidate) (routing.AttemptResult, error) {
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
}

func (s *streamState) ensureHeaders() error {
	if s.started {
		return nil
	}
	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	s.writer.Header().Set("Connection", "keep-alive")
	s.writer.Header().Set("X-Accel-Buffering", "no")
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

func newCompletionID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "chatcmpl_ds_fallback"
	}
	return "chatcmpl_ds_" + hex.EncodeToString(buffer)
}
