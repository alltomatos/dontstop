package provider_test

import (
	"encoding/json"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

func TestChatMessageAcceptsScalarContent(t *testing.T) {
	var message provider.ChatMessage
	if err := json.Unmarshal([]byte(`{"role":"user","content":"ping"}`), &message); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if message.Content != "ping" || message.Parts != nil {
		t.Fatalf("unexpected message: %+v", message)
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != `{"role":"user","content":"ping"}` {
		t.Fatalf("scalar content must stay a string, got %s", encoded)
	}
}

func TestChatMessageAcceptsNullContent(t *testing.T) {
	var message provider.ChatMessage
	if err := json.Unmarshal([]byte(`{"role":"assistant","content":null}`), &message); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if message.Content != "" || message.Parts != nil {
		t.Fatalf("unexpected message: %+v", message)
	}
}

func TestChatMessageAcceptsMultipartContent(t *testing.T) {
	raw := `{"role":"user","content":[
		{"type":"text","text":"describe"},
		{"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA","detail":"high"}},
		{"type":"input_audio","input_audio":{"data":"BBBB","format":"wav"}},
		{"type":"file","file":{"filename":"spec.pdf","file_data":"data:application/pdf;base64,CCCC"}}
	]}`
	var message provider.ChatMessage
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(message.Parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(message.Parts))
	}
	if message.Content != "describe" {
		t.Fatalf("text content not joined: %q", message.Content)
	}
	if message.Parts[1].Modality() != provider.ModalityImage {
		t.Fatalf("unexpected modality %q", message.Parts[1].Modality())
	}
	if message.Parts[2].Modality() != provider.ModalityAudio {
		t.Fatalf("unexpected modality %q", message.Parts[2].Modality())
	}
	if message.Parts[3].Modality() != provider.ModalityPDF {
		t.Fatalf("unexpected modality %q", message.Parts[3].Modality())
	}
	if !message.HasMedia() || len(message.MediaParts()) != 3 {
		t.Fatalf("media detection failed: %+v", message)
	}

	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundtrip map[string]any
	if err := json.Unmarshal(encoded, &roundtrip); err != nil {
		t.Fatalf("decode roundtrip: %v", err)
	}
	if _, ok := roundtrip["content"].([]any); !ok {
		t.Fatalf("multipart content must stay an array, got %s", encoded)
	}
}

func TestChatMessageRejectsInvalidParts(t *testing.T) {
	cases := []string{
		`{"role":"user","content":[{"type":"image_url"}]}`,
		`{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"x","format":"flac"}}]}`,
		`{"role":"user","content":[{"type":"file","file":{}}]}`,
		`{"role":"user","content":[{"type":"video"}]}`,
		`{"role":"user","content":123}`,
	}
	for _, raw := range cases {
		var message provider.ChatMessage
		if err := json.Unmarshal([]byte(raw), &message); err == nil {
			t.Fatalf("expected validation error for %s", raw)
		}
	}
}

func TestParseModalities(t *testing.T) {
	modalities, ok := provider.ParseModalities([]byte(`{"text_input":true,"image_input":true,"audio_input":false,"video_input":false}`))
	if !ok {
		t.Fatal("expected modalities")
	}
	if !modalities.Text || !modalities.Image || modalities.Audio || modalities.Video || !modalities.PDF {
		t.Fatalf("unexpected modalities: %+v", modalities)
	}
	if got := modalities.Inputs(); len(got) != 3 {
		t.Fatalf("unexpected inputs: %v", got)
	}
	if _, ok := provider.ParseModalities([]byte(`{"context_window":128000}`)); ok {
		t.Fatal("capabilities without modality keys must be unknown")
	}
	if _, ok := provider.ParseModalities(nil); ok {
		t.Fatal("empty capabilities must be unknown")
	}
}

func TestLimitToModalitiesDegradesUnsupportedMedia(t *testing.T) {
	var message provider.ChatMessage
	raw := `{"role":"user","content":[
		{"type":"text","text":"look"},
		{"type":"image_url","image_url":{"url":"https://example.com/a.png"}},
		{"type":"file","file":{"filename":"spec.pdf","file_data":"data:application/pdf;base64,CCCC"}}
	]}`
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	limited, dropped := message.LimitToModalities(provider.Modalities{Text: true})
	if len(dropped) != 2 || dropped[0] != provider.ModalityImage || dropped[1] != provider.ModalityPDF {
		t.Fatalf("unexpected dropped modalities: %v", dropped)
	}
	if len(limited.MediaParts()) != 0 {
		t.Fatalf("media must be removed: %+v", limited.Parts)
	}
	if limited.Content == "look" || limited.Content == "" {
		t.Fatalf("expected degradation notes, got %q", limited.Content)
	}

	kept, dropped := message.LimitToModalities(provider.Modalities{Text: true, Image: true, PDF: true})
	if len(dropped) != 0 || len(kept.MediaParts()) != 2 {
		t.Fatalf("supported media must be preserved: dropped=%v parts=%+v", dropped, kept.Parts)
	}
}

func TestLimitToModalitiesLeavesScalarMessages(t *testing.T) {
	message := provider.ChatMessage{Role: "user", Content: "ping"}
	limited, dropped := message.LimitToModalities(provider.Modalities{Text: true})
	if len(dropped) != 0 || limited.Content != "ping" || limited.Parts != nil {
		t.Fatalf("scalar message changed: %+v dropped=%v", limited, dropped)
	}
}
