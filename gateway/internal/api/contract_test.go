package api_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractP0Surface(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "contracts", "openapi-v1.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %v", doc["openapi"])
	}

	components := object(t, doc, "components")
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("missing paths")
	}
	models := object(t, paths, "/models")
	if models["get"] == nil {
		t.Fatal("missing GET /models")
	}
	chat := object(t, paths, "/chat/completions")
	post := object(t, chat, "post")
	if post["operationId"] != "createChatCompletion" {
		t.Fatalf("unexpected operationId: %v", post["operationId"])
	}

	body := object(t, post, "requestBody")
	content := object(t, body, "content")
	jsonContent := object(t, content, "application/json")
	schema := object(t, jsonContent, "schema")
	if ref, ok := schema["$ref"].(string); ok {
		parts := strings.Split(ref, "/")
		schema = object(t, object(t, components, "schemas"), parts[len(parts)-1])
	}
	required := stringSlice(t, schema, "required")
	for _, field := range []string{"model", "messages"} {
		if !contains(required, field) {
			t.Fatalf("request schema must require %s", field)
		}
	}

	responses := object(t, post, "responses")
	response200 := object(t, responses, "200")
	responseContent := object(t, response200, "content")
	if responseContent["text/event-stream"] == nil || responseContent["application/json"] == nil {
		t.Fatal("200 response must expose JSON and SSE")
	}
	for _, status := range []string{"401", "403", "429"} {
		if responses[status] == nil {
			t.Fatalf("missing error response %s", status)
		}
	}

	schemes := object(t, components, "securitySchemes")
	if schemes["bearerAuth"] == nil {
		t.Fatal("missing bearerAuth security scheme")
	}
}

func object(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s must be an object", key)
	}
	return value
}

func stringSlice(t *testing.T, parent map[string]any, key string) []string {
	t.Helper()
	values, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%s must be a list", key)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("%s must contain strings", key)
		}
		result = append(result, text)
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
