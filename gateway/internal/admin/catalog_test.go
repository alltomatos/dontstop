package admin

import "testing"

func TestCatalogAdapterKind(t *testing.T) {
	tests := []struct {
		protocol string
		want     string
		ok       bool
	}{
		{protocol: "openai", want: "openai", ok: true},
		{protocol: "Anthropic", want: "anthropic", ok: true},
		{protocol: "agentrouter", want: "agentrouter", ok: true},
		{protocol: "gemini", ok: false},
		{protocol: "", ok: false},
	}
	for _, test := range tests {
		got, ok := catalogAdapterKind(test.protocol)
		if ok != test.ok || got != test.want {
			t.Fatalf("catalogAdapterKind(%q) = (%q, %v), want (%q, %v)", test.protocol, got, ok, test.want, test.ok)
		}
	}
}

func TestValidateProviderBaseURL(t *testing.T) {
	valid := []string{
		"https://api.openai.com/v1",
		"http://127.0.0.1:8081/v1",
	}
	for _, raw := range valid {
		if err := validateProviderBaseURL(raw); err != nil {
			t.Fatalf("validateProviderBaseURL(%q) returned %v", raw, err)
		}
	}
	invalid := []string{"", "api.openai.com", "ftp://example.com", "https://"}
	for _, raw := range invalid {
		if err := validateProviderBaseURL(raw); err == nil {
			t.Fatalf("validateProviderBaseURL(%q) accepted an invalid URL", raw)
		}
	}
}

func TestJoinModelsURL(t *testing.T) {
	tests := []struct {
		base string
		path string
		want string
	}{
		{base: "https://openrouter.ai/api/v1/", path: "/models", want: "https://openrouter.ai/api/v1/models"},
		{base: "https://api.anthropic.com", path: "v1/models", want: "https://api.anthropic.com/v1/models"},
		{base: "https://api.perplexity.ai", path: "", want: "https://api.perplexity.ai"},
	}
	for _, test := range tests {
		if got := joinModelsURL(test.base, test.path); got != test.want {
			t.Fatalf("joinModelsURL(%q, %q) = %q, want %q", test.base, test.path, got, test.want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	valid := []string{
		"00000000-0000-0000-0000-000000000000",
		"FE54631B-2717-4C3B-ADC4-0F5683665190",
	}
	for _, value := range valid {
		if !isUUID(value) {
			t.Fatalf("isUUID(%q) = false, want true", value)
		}
	}
	invalid := []string{"", "openrouter", "fe54631b-2717-4c3b-adc4", "fe54631b-2717-4c3b-adc4-0f568366519g"}
	for _, value := range invalid {
		if isUUID(value) {
			t.Fatalf("isUUID(%q) = true, want false", value)
		}
	}
}
