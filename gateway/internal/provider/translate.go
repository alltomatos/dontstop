package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrTranslation identifies a prompt or response translation failure.
var ErrTranslation = errors.New("translation failed")

const maxTranslationResponseBytes = 1 << 20

// Translation is one translated text plus the detected source language.
type Translation struct {
	Text           string
	SourceLanguage string
}

// Translator translates text to a target language. Implementations must return
// ErrTranslation when the text cannot be translated.
type Translator interface {
	Translate(ctx context.Context, text, target string) (Translation, error)
}

// HTTPTranslator uses Google's keyless endpoint and falls back to MyMemory.
type HTTPTranslator struct {
	client      *http.Client
	googleURL   string
	myMemoryURL string
}

// NewHTTPTranslator creates a translator with the public fallback endpoints.
func NewHTTPTranslator(client *http.Client) *HTTPTranslator {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPTranslator{
		client:      client,
		googleURL:   "https://translate.googleapis.com/translate_a/single",
		myMemoryURL: "https://api.mymemory.translated.net/get",
	}
}

// NewHTTPTranslatorWithEndpoints creates a translator with explicit endpoints.
// It exists for tests and controlled deployments.
func NewHTTPTranslatorWithEndpoints(client *http.Client, googleURL, myMemoryURL string) *HTTPTranslator {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPTranslator{client: client, googleURL: googleURL, myMemoryURL: myMemoryURL}
}

// Translate translates text and reports the source language detected by the
// upstream translator. Empty text is returned unchanged.
func (t *HTTPTranslator) Translate(ctx context.Context, text, target string) (Translation, error) {
	if strings.TrimSpace(text) == "" {
		return Translation{Text: text, SourceLanguage: target}, nil
	}
	if target == "" {
		target = "en"
	}

	translated, source, err := t.google(ctx, text, target)
	if err == nil {
		return Translation{Text: translated, SourceLanguage: source}, nil
	}
	translated, source, fallbackErr := t.myMemory(ctx, text, target)
	if fallbackErr == nil {
		return Translation{Text: translated, SourceLanguage: source}, nil
	}
	return Translation{}, fmt.Errorf("%w: google: %v; mymemory: %v", ErrTranslation, err, fallbackErr)
}

func (t *HTTPTranslator) google(ctx context.Context, text, target string) (string, string, error) {
	query := url.Values{}
	query.Set("client", "gtx")
	query.Set("sl", "auto")
	query.Set("tl", target)
	query.Set("dt", "t")
	query.Set("q", text)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, t.googleURL+"?"+query.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response, err := t.client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("google status %d", response.StatusCode)
	}
	var payload []any
	if err := json.NewDecoder(io.LimitReader(response.Body, maxTranslationResponseBytes)).Decode(&payload); err != nil {
		return "", "", err
	}
	if len(payload) < 3 {
		return "", "", errors.New("google response is missing chunks or source language")
	}
	chunks, ok := payload[0].([]any)
	if !ok {
		return "", "", errors.New("google response has invalid chunks")
	}
	var translated strings.Builder
	for _, item := range chunks {
		parts, ok := item.([]any)
		if !ok || len(parts) == 0 {
			continue
		}
		value, ok := parts[0].(string)
		if ok {
			translated.WriteString(value)
		}
	}
	source, _ := payload[2].(string)
	if translated.Len() == 0 {
		return "", "", errors.New("google returned an empty translation")
	}
	return translated.String(), source, nil
}

func (t *HTTPTranslator) myMemory(ctx context.Context, text, target string) (string, string, error) {
	if len(text) > 490 {
		return "", "", errors.New("text exceeds MyMemory limit")
	}
	query := url.Values{}
	query.Set("q", text)
	query.Set("langpair", "auto|"+target)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, t.myMemoryURL+"?"+query.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response, err := t.client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("mymemory status %d", response.StatusCode)
	}
	var payload struct {
		ResponseData struct {
			TranslatedText   string `json:"translatedText"`
			DetectedLanguage string `json:"detectedLanguage"`
		} `json:"responseData"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxTranslationResponseBytes)).Decode(&payload); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(payload.ResponseData.TranslatedText) == "" {
		return "", "", errors.New("mymemory returned an empty translation")
	}
	source := payload.ResponseData.DetectedLanguage
	if source == "" {
		source = "auto"
	}
	return payload.ResponseData.TranslatedText, source, nil
}
