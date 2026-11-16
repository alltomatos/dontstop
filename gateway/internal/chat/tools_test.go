package chat_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type toolAdapter struct{}

func (toolAdapter) Chat(context.Context, provider.Account, provider.ChatRequest) (provider.ChatResponse, error) {
	return provider.ChatResponse{ID: "chat-1", Content: "ok"}, nil
}

func (toolAdapter) ChatStream(context.Context, provider.Account, provider.ChatRequest) (provider.Stream, error) {
	return &toolStream{chunks: []provider.StreamChunk{
		{ToolCalls: []provider.ToolCallDelta{{Index: 0, ID: "call_1", Type: "function", Function: provider.ToolCallDeltaFn{Name: "ping"}}}},
		{ToolCalls: []provider.ToolCallDelta{{Index: 0, Function: provider.ToolCallDeltaFn{Arguments: `{"x":1}`}}}},
		{FinishReason: "tool_calls"},
	}}, nil
}

type toolStream struct {
	chunks []provider.StreamChunk
	index  int
}

func (s *toolStream) Recv(context.Context) (provider.StreamChunk, error) {
	if s.index >= len(s.chunks) {
		return provider.StreamChunk{}, io.EOF
	}
	chunk := s.chunks[s.index]
	s.index++
	return chunk, nil
}

func (s *toolStream) Close() error { return nil }

func TestStreamEmitsToolCallDeltas(t *testing.T) {
	service := chat.NewService(
		fakeRouter{candidate: routing.Candidate{AccountID: "account-1", ProviderKind: "openai", BaseURL: "https://provider.example", CredentialCipher: []byte("cipher")}},
		fakeDecrypter{},
		func(string) (provider.Adapter, error) { return toolAdapter{}, nil },
		nil, nil, "v1", nil,
	)
	recorder := httptest.NewRecorder()
	err := service.Stream(context.Background(), "tenant-1", provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "ping"}},
		Tools: []provider.Tool{{Type: "function", Function: provider.ToolFunction{Name: "ping"}}},
	}, recorder)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"tool_calls"`) || !strings.Contains(body, `"ping"`) || !strings.Contains(body, `"arguments"`) {
		t.Fatalf("tool call deltas missing from stream: %s", body)
	}
	if !strings.Contains(body, `"finish_reason":"tool_calls"`) {
		t.Fatalf("finish reason missing from stream: %s", body)
	}
}
