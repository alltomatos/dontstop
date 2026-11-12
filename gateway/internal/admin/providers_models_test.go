package admin

import "testing"

func TestModelsURL(t *testing.T) {
	tests := []struct {
		kind    string
		baseURL string
		want    string
	}{
		{kind: "openai", baseURL: "https://api.openai.com/v1", want: "https://api.openai.com/v1/models"},
		{kind: "openai", baseURL: "https://openrouter.ai/api/v1/", want: "https://openrouter.ai/api/v1/models"},
		{kind: "anthropic", baseURL: "https://api.anthropic.com", want: "https://api.anthropic.com/v1/models"},
		{kind: "anthropic", baseURL: "https://api.anthropic.com/v1", want: "https://api.anthropic.com/v1/models"},
	}
	for _, test := range tests {
		if got := modelsURL(test.kind, test.baseURL); got != test.want {
			t.Fatalf("modelsURL(%q, %q) = %q, want %q", test.kind, test.baseURL, got, test.want)
		}
	}
}
