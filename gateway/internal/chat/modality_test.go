package chat_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/chat"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type capabilityRouter struct{ candidate routing.Candidate }

func (r capabilityRouter) Route(ctx context.Context, _, _ string, attempt routing.Attempt) (routing.Candidate, error) {
	if _, err := attempt(ctx, r.candidate); err != nil {
		return routing.Candidate{}, err
	}
	return r.candidate, nil
}

type capturingAdapter struct {
	requests []provider.ChatRequest
}

func (a *capturingAdapter) Chat(_ context.Context, _ provider.Account, request provider.ChatRequest) (provider.ChatResponse, error) {
	a.requests = append(a.requests, request)
	return provider.ChatResponse{ID: "chat-1", Content: "ok"}, nil
}

func (a *capturingAdapter) ChatStream(_ context.Context, _ provider.Account, request provider.ChatRequest) (provider.Stream, error) {
	a.requests = append(a.requests, request)
	return &fakeStream{}, nil
}

func multimodalRequest() provider.ChatRequest {
	return provider.ChatRequest{
		Model: "gpt-4.1",
		Messages: []provider.ChatMessage{{
			Role: "user",
			Parts: []provider.ContentPart{
				{Type: provider.ContentPartText, Text: "describe"},
				{Type: provider.ContentPartImageURL, ImageURL: &provider.ImageURL{URL: "https://example.com/a.png"}},
			},
		}},
	}
}

func capabilityService(adapter provider.Adapter, modalities *provider.Modalities) *chat.Service {
	return chat.NewService(
		capabilityRouter{candidate: routing.Candidate{
			TenantID:         "tenant-1",
			AccountID:        "account-1",
			ProviderName:     "openai",
			ProviderKind:     "openai",
			ModelID:          "gpt-4.1",
			BaseURL:          "https://provider.example",
			CredentialCipher: []byte("cipher"),
			Capabilities:     modalities,
		}},
		fakeDecrypter{},
		func(string) (provider.Adapter, error) { return adapter, nil },
		nil, nil, "v1", nil,
	)
}

func TestChatDegradesMediaWhenModelIsTextOnly(t *testing.T) {
	adapter := &capturingAdapter{}
	service := capabilityService(adapter, &provider.Modalities{Text: true})

	if _, _, err := service.Chat(context.Background(), "tenant-1", multimodalRequest(), ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	sent := adapter.requests[0].Messages[0]
	if len(sent.MediaParts()) != 0 {
		t.Fatalf("text-only model must not receive media: %+v", sent.Parts)
	}
	if !strings.Contains(sent.Content, "attachment omitted") {
		t.Fatalf("expected degradation note, got %q", sent.Content)
	}
}

func TestChatKeepsMediaWhenModelSupportsIt(t *testing.T) {
	adapter := &capturingAdapter{}
	service := capabilityService(adapter, &provider.Modalities{Text: true, Image: true})

	if _, _, err := service.Chat(context.Background(), "tenant-1", multimodalRequest(), ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	sent := adapter.requests[0].Messages[0]
	if len(sent.MediaParts()) != 1 {
		t.Fatalf("image-capable model must receive media: %+v", sent.Parts)
	}
}

func TestChatPassesMediaThroughWhenCapabilitiesUnknown(t *testing.T) {
	adapter := &capturingAdapter{}
	service := capabilityService(adapter, nil)

	if _, _, err := service.Chat(context.Background(), "tenant-1", multimodalRequest(), ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	sent := adapter.requests[0].Messages[0]
	if len(sent.MediaParts()) != 1 {
		t.Fatalf("unknown capabilities must not drop media: %+v", sent.Parts)
	}
}

func TestChatStreamDegradesMediaWhenModelIsTextOnly(t *testing.T) {
	adapter := &capturingAdapter{}
	service := capabilityService(adapter, &provider.Modalities{Text: true})

	state := httptest.NewRecorder()
	if err := service.Stream(context.Background(), "tenant-1", multimodalRequest(), state); err != nil {
		t.Fatalf("stream: %v", err)
	}
	sent := adapter.requests[0].Messages[0]
	if len(sent.MediaParts()) != 0 || !strings.Contains(sent.Content, "attachment omitted") {
		t.Fatalf("stream must degrade media: %+v", sent)
	}
}
