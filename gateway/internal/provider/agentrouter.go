package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const defaultAgentRouterBaseURL = "https://agentrouter.org"

// agentRouterAdapter translates prompts to English before sending them to
// AgentRouter and translates the response back to the prompt language.
type agentRouterAdapter struct {
	inner      *anthropicAdapter
	translator Translator
}

func newAgentRouterAdapter(client *http.Client, translator Translator) *agentRouterAdapter {
	return &agentRouterAdapter{
		inner:      &anthropicAdapter{client: client},
		translator: translator,
	}
}

func (a *agentRouterAdapter) Chat(ctx context.Context, account Account, request ChatRequest) (ChatResponse, error) {
	translated, source, err := a.translatePrompt(ctx, request)
	if err != nil {
		return ChatResponse{}, err
	}
	response, err := a.inner.Chat(ctx, normalizeAgentRouterAccount(account), translated)
	if err != nil {
		return ChatResponse{}, err
	}
	response.Content, err = a.translateResponse(ctx, response.Content, source)
	if err != nil {
		return ChatResponse{}, err
	}
	return response, nil
}

func (a *agentRouterAdapter) ChatStream(ctx context.Context, account Account, request ChatRequest) (Stream, error) {
	translated, source, err := a.translatePrompt(ctx, request)
	if err != nil {
		return nil, err
	}
	stream, err := a.inner.ChatStream(ctx, normalizeAgentRouterAccount(account), translated)
	if err != nil {
		return nil, err
	}
	return &translatedStream{
		inner:      stream,
		translator: a.translator,
		target:     source,
	}, nil
}

func normalizeAgentRouterAccount(account Account) Account {
	if strings.TrimSpace(account.BaseURL) == "" {
		account.BaseURL = defaultAgentRouterBaseURL
	}
	return account
}

func (a *agentRouterAdapter) translatePrompt(ctx context.Context, request ChatRequest) (ChatRequest, string, error) {
	if a.translator == nil {
		return ChatRequest{}, "", fmt.Errorf("%w: translator is not configured", ErrTranslation)
	}
	translated := request
	translated.Messages = make([]ChatMessage, len(request.Messages))
	source := "en"
	for index, message := range request.Messages {
		translated.Messages[index] = message
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		result, err := a.translator.Translate(ctx, message.Content, "en")
		if err != nil {
			slog.Warn("agentrouter prompt translation failed", "stage", "prompt", "reason", err.Error())
			return ChatRequest{}, "", fmt.Errorf("%w: prompt: %v", ErrTranslation, err)
		}
		translated.Messages[index].Content = result.Text
		if message.Role == "user" && result.SourceLanguage != "" {
			source = result.SourceLanguage
		}
	}
	return translated, source, nil
}

func (a *agentRouterAdapter) translateResponse(ctx context.Context, text, source string) (string, error) {
	if strings.TrimSpace(text) == "" || source == "" || source == "en" {
		return text, nil
	}
	if source == "auto" {
		err := fmt.Errorf("%w: response source language is unknown", ErrTranslation)
		slog.Warn("agentrouter response translation failed", "stage", "response", "reason", err.Error())
		return "", err
	}
	result, err := a.translator.Translate(ctx, text, source)
	if err != nil {
		slog.Warn("agentrouter response translation failed", "stage", "response", "reason", err.Error())
		return "", fmt.Errorf("%w: response: %v", ErrTranslation, err)
	}
	return result.Text, nil
}

type translatedStream struct {
	inner      Stream
	translator Translator
	target     string
	content    strings.Builder
	finish     string
	usage      *Usage
	done       bool
}

func (s *translatedStream) Recv(ctx context.Context) (StreamChunk, error) {
	if s.done {
		return StreamChunk{}, io.EOF
	}
	for {
		chunk, err := s.inner.Recv(ctx)
		if errors.Is(err, io.EOF) {
			s.done = true
			text := s.content.String()
			if strings.TrimSpace(text) != "" && s.target != "" && s.target != "en" {
				if s.target == "auto" {
					return StreamChunk{}, fmt.Errorf("%w: stream source language is unknown", ErrTranslation)
				}
				result, translateErr := s.translator.Translate(ctx, text, s.target)
				if translateErr != nil {
					slog.Warn("agentrouter stream translation failed", "stage", "stream", "reason", translateErr.Error())
					return StreamChunk{}, fmt.Errorf("%w: stream: %v", ErrTranslation, translateErr)
				}
				text = result.Text
			}
			return StreamChunk{Content: text, FinishReason: s.finish, Usage: s.usage}, nil
		}
		if err != nil {
			return StreamChunk{}, err
		}
		s.content.WriteString(chunk.Content)
		if chunk.FinishReason != "" {
			s.finish = chunk.FinishReason
		}
		if chunk.Usage != nil {
			s.usage = chunk.Usage
		}
	}
}

func (s *translatedStream) Close() error {
	return s.inner.Close()
}
