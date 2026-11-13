package provider_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

// TestAgentRouterLive is opt-in. It requires a valid AgentRouter key and a
// provider base URL. The adapter is native; no external gateway or relay is used.
func TestAgentRouterLive(t *testing.T) {
	key := os.Getenv("AGENTROUTER_LIVE_KEY")
	baseURL := os.Getenv("AGENTROUTER_LIVE_BASE_URL")
	if key == "" || baseURL == "" {
		t.Skip("AGENTROUTER_LIVE_KEY and AGENTROUTER_LIVE_BASE_URL are not set")
	}
	model := os.Getenv("AGENTROUTER_LIVE_MODEL")
	if model == "" {
		model = "gpt-5.6-sol"
	}
	adapter, err := provider.NewAdapter("agentrouter")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	response, err := adapter.Chat(ctx, provider.Account{BaseURL: baseURL, APIKey: key}, provider.ChatRequest{
		Model:     model,
		MaxTokens: 16,
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Reply with the single word: ok"},
		},
	})
	if err != nil {
		t.Fatalf("live chat: %v", err)
	}
	if response.Content == "" {
		t.Fatal("live response is empty")
	}
	t.Logf("live response: %q", response.Content)
}
