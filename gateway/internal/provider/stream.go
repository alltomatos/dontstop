package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamChunk is a provider-neutral streaming delta.
type StreamChunk struct {
	Content      string
	FinishReason string
	Usage        *Usage
}

// Stream yields provider-neutral chunks until io.EOF.
type Stream interface {
	Recv(ctx context.Context) (StreamChunk, error)
	Close() error
}

type openAIStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	model   string
}

func newOpenAIStream(body io.ReadCloser, model string) *openAIStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	return &openAIStream{body: body, scanner: scanner, model: model}
}

func (s *openAIStream) Recv(_ context.Context) (StreamChunk, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return StreamChunk{}, io.EOF
		}
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *Usage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return StreamChunk{}, fmt.Errorf("%w: %v", ErrProviderResponse, err)
		}
		result := StreamChunk{}
		if len(chunk.Choices) > 0 {
			result.Content = chunk.Choices[0].Delta.Content
			result.FinishReason = normalizeFinishReason(chunk.Choices[0].FinishReason)
			if chunk.Choices[0].FinishReason == "" {
				result.FinishReason = ""
			}
		}
		if chunk.Usage != nil {
			usage := *chunk.Usage
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			result.Usage = &usage
		}
		if result.Content == "" && result.FinishReason == "" && result.Usage == nil {
			continue
		}
		return result, nil
	}
	if err := s.scanner.Err(); err != nil {
		return StreamChunk{}, fmt.Errorf("%w: %v", ErrProviderResponse, err)
	}
	return StreamChunk{}, io.EOF
}

func (s *openAIStream) Close() error {
	return s.body.Close()
}

type anthropicStream struct {
	body         io.ReadCloser
	scanner      *bufio.Scanner
	inputTokens  int
	outputTokens int
}

func newAnthropicStream(body io.ReadCloser) *anthropicStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	return &anthropicStream{body: body, scanner: scanner}
}

func (s *anthropicStream) Recv(_ context.Context) (StreamChunk, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
			Message struct {
				Usage struct {
					InputTokens int `json:"input_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return StreamChunk{}, fmt.Errorf("%w: %v", ErrProviderResponse, err)
		}
		switch event.Type {
		case "message_start":
			s.inputTokens = event.Message.Usage.InputTokens
		case "content_block_delta":
			if event.Delta.Text != "" {
				return StreamChunk{Content: event.Delta.Text}, nil
			}
		case "message_delta":
			s.outputTokens = event.Usage.OutputTokens
			return StreamChunk{
				FinishReason: normalizeFinishReason(event.Delta.StopReason),
				Usage: &Usage{
					PromptTokens:     s.inputTokens,
					CompletionTokens: s.outputTokens,
					TotalTokens:      s.inputTokens + s.outputTokens,
				},
			}, nil
		case "message_stop":
			return StreamChunk{}, io.EOF
		}
	}
	if err := s.scanner.Err(); err != nil {
		return StreamChunk{}, fmt.Errorf("%w: %v", ErrProviderResponse, err)
	}
	return StreamChunk{}, io.EOF
}

func (s *anthropicStream) Close() error {
	return s.body.Close()
}
