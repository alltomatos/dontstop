package provider

import (
	"context"
	"net/http"
	"strings"
)

type anthropicAdapter struct {
	client *http.Client
}

type anthropicRequest struct {
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	System      string        `json:"system,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
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
	messages := make([]ChatMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		if message.Role == "system" {
			system = append(system, message.Content)
			continue
		}
		messages = append(messages, message)
	}
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
	for _, block := range response.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}
	return ChatResponse{
		ID:           response.ID,
		Model:        response.Model,
		Content:      content.String(),
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
	messages := make([]ChatMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		if message.Role == "system" {
			system = append(system, message.Content)
			continue
		}
		messages = append(messages, message)
	}
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
