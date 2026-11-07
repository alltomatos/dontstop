package provider_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func TestHTTPTranslatorUsesGoogleAndReportsSourceLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("tl") != "en" {
			t.Fatalf("unexpected target: %s", request.URL.Query().Get("tl"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[[["Hello","Olá",null,null,10]],null,"pt",null,null,null,1.0]`))
	}))
	defer server.Close()

	translator := provider.NewHTTPTranslatorWithEndpoints(server.Client(), server.URL, server.URL)
	result, err := translator.Translate(context.Background(), "Olá", "en")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if result.Text != "Hello" || result.SourceLanguage != "pt" {
		t.Fatalf("unexpected translation: %+v", result)
	}
}

func TestHTTPTranslatorFallsBackToMyMemory(t *testing.T) {
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer google.Close()
	memory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("q") != "Olá" {
			t.Fatalf("unexpected query: %s", request.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseData":{"translatedText":"Hello","detectedLanguage":"pt"}}`))
	}))
	defer memory.Close()

	translator := provider.NewHTTPTranslatorWithEndpoints(google.Client(), google.URL, memory.URL)
	result, err := translator.Translate(context.Background(), "Olá", "en")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if result.Text != "Hello" || result.SourceLanguage != "pt" {
		t.Fatalf("unexpected fallback translation: %+v", result)
	}
}

func TestHTTPTranslatorFailsClosedWhenBothEndpointsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	translator := provider.NewHTTPTranslatorWithEndpoints(server.Client(), server.URL, server.URL)
	_, err := translator.Translate(context.Background(), "Olá", "en")
	if !errors.Is(err, provider.ErrTranslation) {
		t.Fatalf("expected ErrTranslation, got %v", err)
	}
}
