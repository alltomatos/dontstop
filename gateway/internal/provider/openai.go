package provider

import (
	"context"
	"net/http"
	"strings"
)

type openAIAdapter struct {
	client *http.Client
}

type openAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

func (a *openAIAdapter) Chat(ctx context.Context, account Account, request ChatRequest) (ChatResponse, error) {
	baseURL := strings.TrimRight(account.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	payload := openAIRequest{
		Model:       request.Model,
		Messages:    request.Messages,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
	}
	var response openAIResponse
	err := postJSON(ctx, clientFor(a.client, account), baseURL+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + account.APIKey,
	}, payload, &response, account.ProxyURL != "")
	if err != nil {
		return ChatResponse{}, err
	}
	if len(response.Choices) == 0 {
		return ChatResponse{}, ErrProviderResponse
	}
	response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
	return ChatResponse{
		ID:           response.ID,
		Model:        response.Model,
		Content:      response.Choices[0].Message.Content,
		FinishReason: normalizeFinishReason(response.Choices[0].FinishReason),
		Usage:        response.Usage,
	}, nil
}

func normalizeFinishReason(reason string) string {
	switch reason {
	case "length", "max_tokens":
		return "length"
	case "tool_calls", "tool_use":
		return "tool_calls"
	case "content_filter":
		return "content_filter"
	default:
		return "stop"
	}
}
func (a *openAIAdapter) ChatStream(ctx context.Context, account Account, request ChatRequest) (Stream, error) {
	baseURL := strings.TrimRight(account.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	payload := openAIRequest{
		Model:       request.Model,
		Messages:    request.Messages,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
		Stream:      true,
	}
	body, err := postSSE(ctx, clientFor(a.client, account), baseURL+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + account.APIKey,
	}, payload, account.ProxyURL != "")
	if err != nil {
		return nil, err
	}
	return newOpenAIStream(body, request.Model), nil
}
