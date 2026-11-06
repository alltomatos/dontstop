package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

// ChatService executes chat completions for the HTTP layer.
type ChatService interface {
	Chat(ctx context.Context, tenantID string, request provider.ChatRequest, idempotencyKey string) (provider.ChatResponse, string, error)
	Stream(ctx context.Context, tenantID string, request provider.ChatRequest, writer http.ResponseWriter) error
}

func chatCompletions(service ChatService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok || !principal.HasScope("chat:write") {
			writeAPIError(w, http.StatusForbidden, "permission_error", "missing chat:write scope")
			return
		}

		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var payload provider.ChatRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		if err := validateChatRequest(payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
			return
		}

		if payload.Stream {
			if err := service.Stream(request.Context(), principal.TenantID, payload, w); err != nil {
				if w.Header().Get("Content-Type") != "text/event-stream" {
					writeChatError(w, err)
				}
			}
			return
		}

		response, cacheStatus, err := service.Chat(request.Context(), principal.TenantID, payload, request.Header.Get("Idempotency-Key"))
		if err != nil {
			writeChatError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Donstop-Cache", cacheStatus)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      response.ID,
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   response.Model,
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": response.Content,
				},
				"finish_reason": response.FinishReason,
			}},
			"usage": map[string]int{
				"prompt_tokens":     response.Usage.PromptTokens,
				"completion_tokens": response.Usage.CompletionTokens,
				"total_tokens":      response.Usage.TotalTokens,
			},
		})
	}
}

func validateChatRequest(request provider.ChatRequest) error {
	if request.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return fmt.Errorf("messages must contain at least one item")
	}
	for _, message := range request.Messages {
		switch message.Role {
		case "system", "user", "assistant", "tool":
		default:
			return fmt.Errorf("invalid message role %q", message.Role)
		}
	}
	if request.Temperature != nil && (*request.Temperature < 0 || *request.Temperature > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if request.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	return nil
}

func writeChatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, routing.ErrNoCandidate):
		writeAPIError(w, http.StatusServiceUnavailable, "model_unavailable", "no account available for the requested model")
	case errors.Is(err, routing.ErrAllCandidatesFailed):
		if errors.Is(err, provider.ErrProviderRateLimited) {
			writeAPIError(w, http.StatusTooManyRequests, "rate_limit_error", "upstream rate limit")
			return
		}
		writeAPIError(w, http.StatusBadGateway, "api_error", "upstream provider failed")
	case errors.Is(err, context.DeadlineExceeded):
		writeAPIError(w, http.StatusGatewayTimeout, "timeout_error", "upstream timeout")
	default:
		writeAPIError(w, http.StatusBadGateway, "api_error", "upstream request failed")
	}
}
