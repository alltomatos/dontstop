package chat

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

// cacheStatusHit, cacheStatusMiss and cacheStatusBypass are the values exposed
// in X-Donstop-Cache and persisted in requests.cache_status.
const (
	cacheStatusHit    = "hit"
	cacheStatusMiss   = "miss"
	cacheStatusBypass = "bypass"
)

// buildCacheKey returns the tenant-scoped exact cache key for a request. An
// empty string means the key cannot be derived and the request must not be cached.
func (s *Service) buildCacheKey(tenantID string, request provider.ChatRequest) string {
	key, err := cache.BuildKey(cache.KeyInput{
		TenantID: tenantID, Scope: "chat:write", Model: request.Model,
		PromptVersion: s.promptVersion, Messages: request.Messages,
		Tools: request.Tools, ToolChoice: request.ToolChoice,
		Temperature: request.Temperature, MaxTokens: request.MaxTokens,
	})
	if err != nil {
		return ""
	}
	return key
}

// replayCachedStream serves a streamed request from the exact cache. The stored
// response is replayed as SSE chunks so the client contract stays identical to
// an upstream stream. It reports whether the request was fully handled.
func (s *Service) replayCachedStream(ctx context.Context, tenantID string, request provider.ChatRequest, cacheKey string, state *streamState, started time.Time) (bool, error) {
	if s.cache == nil || cacheKey == "" {
		state.cacheStatus = cacheStatusBypass
		return false, nil
	}
	entry, found, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		state.cacheStatus = cacheStatusBypass
		return false, nil
	}
	if !found || entry.ErrorCode != "" {
		state.cacheStatus = cacheStatusMiss
		return false, nil
	}
	state.cacheStatus = cacheStatusHit
	s.recordUsage(ctx, tenantID, "", "", request.Model, "success", cacheStatusHit, entry.Response.Usage, time.Since(started))
	return true, state.replay(entry.Response)
}

// storeStreamResponse caches the aggregated stream so the next identical request
// can be served without reaching a provider.
func (s *Service) storeStreamResponse(ctx context.Context, tenantID string, selected routing.Candidate, request provider.ChatRequest, cacheKey string, state *streamState) {
	if s.cache == nil || cacheKey == "" || !state.started || state.failed {
		return
	}
	response, ok := state.cachedResponse()
	if !ok {
		return
	}
	_ = s.cache.Set(ctx, cacheKey, cache.Meta{
		TenantID: tenantID, Model: request.Model, Provider: selected.ProviderName,
		PromptVersion: s.promptVersion,
	}, cache.Entry{Response: response}, time.Hour)
}

// cachedResponse rebuilds a provider-neutral response from the emitted deltas.
// It refuses to cache empty completions so a broken upstream cannot poison the
// cache with an empty answer.
func (s *streamState) cachedResponse() (provider.ChatResponse, bool) {
	content := s.content.String()
	toolCalls := s.assembledToolCalls()
	if strings.TrimSpace(content) == "" && len(toolCalls) == 0 {
		return provider.ChatResponse{}, false
	}
	return provider.ChatResponse{
		ID:           s.id,
		Model:        s.model,
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: s.finishReason,
		Usage:        s.usage,
	}, true
}

// recordToolCalls accumulates streaming tool-call fragments so a cached replay
// can reproduce the completed calls. Fragments arrive split by index and must be
// concatenated, not overwritten.
func (s *streamState) recordToolCalls(deltas []provider.ToolCallDelta) {
	if len(deltas) == 0 {
		return
	}
	if s.toolCalls == nil {
		s.toolCalls = make(map[int]*provider.ToolCall)
	}
	for _, delta := range deltas {
		call, ok := s.toolCalls[delta.Index]
		if !ok {
			call = &provider.ToolCall{Type: "function"}
			s.toolCalls[delta.Index] = call
		}
		if delta.ID != "" {
			call.ID = delta.ID
		}
		if delta.Type != "" {
			call.Type = delta.Type
		}
		call.Function.Name += delta.Function.Name
		call.Function.Arguments += delta.Function.Arguments
	}
}

// assembledToolCalls merges the streaming fragments collected by index.
func (s *streamState) assembledToolCalls() []provider.ToolCall {
	if len(s.toolCalls) == 0 {
		return nil
	}
	calls := make([]provider.ToolCall, 0, len(s.toolCalls))
	indexes := make([]int, 0, len(s.toolCalls))
	for index := range s.toolCalls {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		calls = append(calls, *s.toolCalls[index])
	}
	return calls
}

// replay writes a cached response as an OpenAI-compatible SSE stream.
func (s *streamState) replay(response provider.ChatResponse) error {
	s.cacheStatus = cacheStatusHit
	if err := s.emitChunk(provider.StreamChunk{Content: response.Content}); err != nil {
		return err
	}
	usage := response.Usage
	finish := response.FinishReason
	if finish == "" {
		finish = "stop"
	}
	if err := s.emitChunk(provider.StreamChunk{FinishReason: finish, Usage: &usage}); err != nil {
		return err
	}
	s.done()
	return nil
}

// emitChunk renders one synthetic chunk without touching the aggregation state,
// because the replayed content was already persisted when it was first cached.
func (s *streamState) emitChunk(chunk provider.StreamChunk) error {
	if err := s.ensureHeaders(); err != nil {
		return err
	}
	delta := map[string]any{}
	if chunk.Content != "" {
		delta["content"] = chunk.Content
	}
	payload := map[string]any{
		"id":      s.id,
		"object":  "chat.completion.chunk",
		"created": s.created,
		"model":   s.model,
		"choices": []map[string]any{{
			"index":         0,
			"delta":         delta,
			"finish_reason": nil,
		}},
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

// normalizeCacheStatus maps an internal status to the public header value.
func normalizeCacheStatus(status string) string {
	switch status {
	case cacheStatusHit, cacheStatusMiss, cacheStatusBypass:
		return status
	default:
		return cacheStatusBypass
	}
}
