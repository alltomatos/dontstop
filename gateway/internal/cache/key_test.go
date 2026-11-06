package cache_test

import (
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/cache"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func baseInput() cache.KeyInput {
	temp := 0.2
	return cache.KeyInput{
		TenantID:      "tenant-1",
		Scope:         "chat:write",
		Model:         "gpt-4.1",
		PromptVersion: "v1",
		Messages:      []provider.ChatMessage{{Role: "user", Content: "Oi"}},
		Temperature:   &temp,
		MaxTokens:     100,
	}
}

func TestBuildKeyIsStableAndTenantScoped(t *testing.T) {
	input := baseInput()
	first, err := cache.BuildKey(input)
	if err != nil {
		t.Fatalf("build key: %v", err)
	}
	second, err := cache.BuildKey(input)
	if err != nil {
		t.Fatalf("build key: %v", err)
	}
	if first != second {
		t.Fatal("same input must build the same key")
	}
	if len(first) < len("t:tenant-1:cache:exact:") || first[:len("t:tenant-1:cache:exact:")] != "t:tenant-1:cache:exact:" {
		t.Fatalf("unexpected key prefix: %s", first)
	}

	changed := input
	changed.TenantID = "tenant-2"
	other, _ := cache.BuildKey(changed)
	if other == first {
		t.Fatal("tenant must change the key")
	}
}

func TestBuildKeyChangesWithModelParamsAndMessages(t *testing.T) {
	input := baseInput()
	base, _ := cache.BuildKey(input)

	for name, mutate := range map[string]func(*cache.KeyInput){
		"model":       func(i *cache.KeyInput) { i.Model = "claude-sonnet-4" },
		"temperature": func(i *cache.KeyInput) { value := 0.7; i.Temperature = &value },
		"max_tokens":  func(i *cache.KeyInput) { i.MaxTokens = 200 },
		"prompt":      func(i *cache.KeyInput) { i.PromptVersion = "v2" },
		"messages":    func(i *cache.KeyInput) { i.Messages = []provider.ChatMessage{{Role: "user", Content: "Outro"}} },
	} {
		changed := input
		mutate(&changed)
		other, err := cache.BuildKey(changed)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if other == base {
			t.Fatalf("%s must change the key", name)
		}
	}
}

func TestBuildKeyRejectsMissingFields(t *testing.T) {
	input := baseInput()
	input.TenantID = ""
	if _, err := cache.BuildKey(input); err == nil {
		t.Fatal("expected tenant error")
	}
	input = baseInput()
	input.Model = ""
	if _, err := cache.BuildKey(input); err == nil {
		t.Fatal("expected model error")
	}
}
