package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type anthropicAdapter struct {
	client *http.Client
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature *float64           `json:"temperature,omitempty"`
	Stream      bool               `json:"stream"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	ToolChoice  *anthropicChoice   `json:"tool_choice,omitempty"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *anthropicAdapter) Chat(ctx context.Context, account Account, request ChatRequest) (ChatResponse, error) {
	baseURL := strings.TrimRight(account.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	system := make([]string, 0)
	messages := anthropicMessages(request.Messages, &system)
	tools, choice := anthropicTools(request)
	maxTokens := request.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	payload := anthropicRequest{
		Model:       request.Model,
		MaxTokens:   maxTokens,
		System:      strings.Join(system, "\n"),
		Messages:    messages,
		Temperature: request.Temperature,
		Tools:       tools,
		ToolChoice:  choice,
	}
	var response anthropicResponse
	err := postJSON(ctx, clientFor(a.client, account), baseURL+"/v1/messages", map[string]string{
		"x-api-key":         account.APIKey,
		"anthropic-version": "2023-06-01",
	}, payload, &response, account.ProxyURL != "")
	if err != nil {
		return ChatResponse{}, err
	}

	var content strings.Builder
	toolCalls := make([]ToolCall, 0)
	for _, block := range response.Content {
		switch block.Type {
		case "text":
			content.WriteString(block.Text)
		case "tool_use":
			arguments := string(block.Input)
			if len(block.Input) == 0 {
				arguments = "{}"
			}
			toolCalls = append(toolCalls, ToolCall{
				ID: block.ID, Type: "function",
				Function: ToolCallFunction{Name: block.Name, Arguments: arguments},
			})
		}
	}
	return ChatResponse{
		ID:           response.ID,
		Model:        response.Model,
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: normalizeFinishReason(response.StopReason),
		Usage: Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
	}, nil
}

func (a *anthropicAdapter) ChatStream(ctx context.Context, account Account, request ChatRequest) (Stream, error) {
	baseURL := strings.TrimRight(account.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	system := make([]string, 0)
	messages := anthropicMessages(request.Messages, &system)
	tools, choice := anthropicTools(request)
	maxTokens := request.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	payload := anthropicRequest{
		Model:       request.Model,
		MaxTokens:   maxTokens,
		System:      strings.Join(system, "\n"),
		Messages:    messages,
		Temperature: request.Temperature,
		Stream:      true,
		Tools:       tools,
		ToolChoice:  choice,
	}
	body, err := postSSE(ctx, clientFor(a.client, account), baseURL+"/v1/messages", map[string]string{
		"x-api-key":         account.APIKey,
		"anthropic-version": "2023-06-01",
	}, payload, account.ProxyURL != "")
	if err != nil {
		return nil, err
	}
	return newAnthropicStream(body), nil
}
