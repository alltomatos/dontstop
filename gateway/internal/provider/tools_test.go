package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func pingTool() provider.Tool {
	return provider.Tool{
		Type: "function",
		Function: provider.ToolFunction{
			Name:        "ping",
			Description: "ping the server",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"]}`),
		},
	}
}

func TestToolValidation(t *testing.T) {
	if err := pingTool().Validate(); err != nil {
		t.Fatalf("valid tool rejected: %v", err)
	}
	bad := []provider.Tool{
		{Type: "other", Function: provider.ToolFunction{Name: "x"}},
		{Type: "function", Function: provider.ToolFunction{}},
		{Type: "function", Function: provider.ToolFunction{Name: "x", Parameters: json.RawMessage(`{`)}},
	}
	for _, tool := range bad {
		if err := tool.Validate(); err == nil {
			t.Fatalf("expected validation error for %+v", tool)
		}
	}
}

func TestValidateToolChoice(t *testing.T) {
	valid := []string{`"auto"`, `"none"`, `"required"`, `{"type":"function","function":{"name":"ping"}}`}
	for _, raw := range valid {
		if err := provider.ValidateToolChoice(json.RawMessage(raw)); err != nil {
			t.Fatalf("valid choice %s rejected: %v", raw, err)
		}
	}
	invalid := []string{`"always"`, `{"type":"function"}`, `42`}
	for _, raw := range invalid {
		if err := provider.ValidateToolChoice(json.RawMessage(raw)); err == nil {
			t.Fatalf("expected validation error for %s", raw)
		}
	}
	if provider.ToolChoiceMode(json.RawMessage(`"required"`)) != "required" {
		t.Fatal("tool choice mode not detected")
	}
	if provider.ToolChoiceName(json.RawMessage(`{"type":"function","function":{"name":"ping"}}`)) != "ping" {
		t.Fatal("tool choice name not detected")
	}
}

func TestChatMessageRoundTripsToolCalls(t *testing.T) {
	raw := `{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"ping","arguments":"{\"x\":1}"}}]}`
	var message provider.ChatMessage
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(message.ToolCalls) != 1 || message.ToolCalls[0].Function.Name != "ping" {
		t.Fatalf("tool calls not decoded: %+v", message)
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"tool_calls"`) {
		t.Fatalf("tool calls not re-encoded: %s", encoded)
	}
}

func TestOpenAIAdapterForwardsToolsAndParsesToolCalls(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"c1","model":"gpt-4.1","choices":[{"message":{"content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"ping","arguments":"{\"x\":1}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("openai")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	response, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "ping"}},
		Tools: []provider.Tool{pingTool()}, ToolChoice: json.RawMessage(`"required"`),
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	tools := received["tools"].([]any)
	if len(tools) != 1 || received["tool_choice"] != "required" {
		t.Fatalf("tools not forwarded: %#v choice=%#v", tools, received["tool_choice"])
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call_1" || response.FinishReason != "tool_calls" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestOpenAIStreamEmitsToolCallDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"ping","arguments":""}}]},"finish_reason":null}]}`,
			``,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"x\":1}"}}]},"finish_reason":null}]}`,
			``,
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("openai")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	stream, err := adapter.ChatStream(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "gpt-4.1", Messages: []provider.ChatMessage{{Role: "user", Content: "ping"}}, Tools: []provider.Tool{pingTool()},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer func() { _ = stream.Close() }()

	first, err := stream.Recv(context.Background())
	if err != nil || len(first.ToolCalls) != 1 || first.ToolCalls[0].Function.Name != "ping" {
		t.Fatalf("unexpected first chunk: %+v err=%v", first, err)
	}
	second, err := stream.Recv(context.Background())
	if err != nil || len(second.ToolCalls) != 1 || second.ToolCalls[0].Function.Arguments != `{"x":1}` {
		t.Fatalf("unexpected second chunk: %+v err=%v", second, err)
	}
	third, err := stream.Recv(context.Background())
	if err != nil || third.FinishReason != "tool_calls" {
		t.Fatalf("unexpected finish chunk: %+v err=%v", third, err)
	}
}

func TestAnthropicAdapterLowersToolsAndParsesToolUse(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m1","model":"claude-sonnet-4","content":[{"type":"tool_use","id":"toolu_1","name":"ping","input":{"x":1}}],"stop_reason":"tool_use","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("anthropic")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	response, err := adapter.Chat(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "ping"},
			{Role: "assistant", ToolCalls: []provider.ToolCall{{
				ID: "toolu_1", Type: "function",
				Function: provider.ToolCallFunction{Name: "ping", Arguments: `{"x":1}`},
			}}},
			{Role: "tool", ToolCallID: "toolu_1", Content: `{"ok":true}`},
		},
		Tools: []provider.Tool{pingTool()}, ToolChoice: json.RawMessage(`"required"`),
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	tools := received["tools"].([]any)
	tool := tools[0].(map[string]any)
	if tool["name"] != "ping" || tool["input_schema"] == nil {
		t.Fatalf("unexpected anthropic tool: %#v", tool)
	}
	if received["tool_choice"].(map[string]any)["type"] != "any" {
		t.Fatalf("unexpected anthropic tool_choice: %#v", received["tool_choice"])
	}

	messages := received["messages"].([]any)
	assistant := messages[1].(map[string]any)["content"].([]any)[0].(map[string]any)
	if assistant["type"] != "tool_use" || assistant["id"] != "toolu_1" || assistant["name"] != "ping" {
		t.Fatalf("unexpected assistant tool_use block: %#v", assistant)
	}
	toolResult := messages[2].(map[string]any)
	resultBlock := toolResult["content"].([]any)[0].(map[string]any)
	if toolResult["role"] != "user" || resultBlock["type"] != "tool_result" || resultBlock["tool_use_id"] != "toolu_1" {
		t.Fatalf("unexpected tool_result message: %#v", toolResult)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].Function.Name != "ping" || response.ToolCalls[0].Function.Arguments != `{"x":1}` {
		t.Fatalf("unexpected response tool calls: %+v", response)
	}
}

func TestAnthropicStreamEmitsToolCallDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"ping"}}`,
			``,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"x\":1}"}}`,
			``,
			`data: {"type":"content_block_stop","index":0}`,
			``,
			`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}`,
			``,
			`data: {"type":"message_stop"}`,
			``,
		}, "\n")))
	}))
	defer server.Close()

	adapter, err := provider.NewAdapter("anthropic")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	stream, err := adapter.ChatStream(context.Background(), provider.Account{BaseURL: server.URL, APIKey: "key"}, provider.ChatRequest{
		Model: "claude-sonnet-4", Messages: []provider.ChatMessage{{Role: "user", Content: "ping"}}, Tools: []provider.Tool{pingTool()},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer func() { _ = stream.Close() }()

	first, err := stream.Recv(context.Background())
	if err != nil || len(first.ToolCalls) != 1 || first.ToolCalls[0].ID != "toolu_1" || first.ToolCalls[0].Function.Name != "ping" {
		t.Fatalf("unexpected first chunk: %+v err=%v", first, err)
	}
	second, err := stream.Recv(context.Background())
	if err != nil || len(second.ToolCalls) != 1 || second.ToolCalls[0].Function.Arguments != `{"x":1}` {
		t.Fatalf("unexpected second chunk: %+v err=%v", second, err)
	}
	third, err := stream.Recv(context.Background())
	if err != nil || third.FinishReason != "tool_calls" {
		t.Fatalf("unexpected finish chunk: %+v err=%v", third, err)
	}
}
