package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

// KeyInput contains the normalized dimensions of an exact cache key.
type KeyInput struct {
	TenantID      string
	Scope         string
	Model         string
	PromptVersion string
	Messages      []provider.ChatMessage
	Temperature   *float64
	MaxTokens     int
}

// BuildKey returns the tenant-prefixed Redis key for an exact response.
func BuildKey(input KeyInput) (string, error) {
	if strings.TrimSpace(input.TenantID) == "" {
		return "", fmt.Errorf("tenant is required")
	}
	if strings.TrimSpace(input.Model) == "" {
		return "", fmt.Errorf("model is required")
	}
	payload, err := json.Marshal(struct {
		Scope         string                 `json:"scope"`
		Model         string                 `json:"model"`
		PromptVersion string                 `json:"prompt_version"`
		Messages      []provider.ChatMessage `json:"messages"`
		Temperature   *float64               `json:"temperature"`
		MaxTokens     int                    `json:"max_tokens"`
	}{
		Scope: input.Scope, Model: input.Model, PromptVersion: input.PromptVersion,
		Messages: input.Messages, Temperature: input.Temperature, MaxTokens: input.MaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("encode cache key: %w", err)
	}
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("t:%s:cache:exact:%s", input.TenantID, hex.EncodeToString(sum[:])), nil
}
