package provider

import (
	"encoding/base64"
	"log/slog"
	"strings"
)

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

type anthropicBlock struct {
	Type   string           `json:"type"`
	Text   string           `json:"text,omitempty"`
	Source *anthropicSource `json:"source,omitempty"`
}

// anthropicMessages splits system prompts out of the message list and lowers
// OpenAI-style content parts into Anthropic blocks. Media the Anthropic message
// API cannot carry is replaced by a text note and logged.
func anthropicMessages(messages []ChatMessage, system *[]string) []anthropicMessage {
	converted := make([]anthropicMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" {
			*system = append(*system, message.Content)
			if message.HasMedia() {
				slog.Warn("anthropic system media dropped", "parts", len(message.MediaParts()))
			}
			continue
		}
		converted = append(converted, anthropicMessage{Role: message.Role, Content: anthropicContent(message)})
	}
	return converted
}

func anthropicContent(message ChatMessage) any {
	if message.Parts == nil {
		return message.Content
	}
	blocks := make([]anthropicBlock, 0, len(message.Parts))
	degraded := make([]string, 0, len(message.Parts))
	for _, part := range message.Parts {
		block, ok := anthropicBlockFor(part)
		if !ok {
			degraded = append(degraded, part.Modality())
			blocks = append(blocks, anthropicBlock{Type: "text", Text: omissionNote(part, part.Modality())})
			continue
		}
		blocks = append(blocks, block)
	}
	if len(degraded) > 0 {
		slog.Warn("anthropic media degraded", "modalities", degraded)
	}
	if len(blocks) == 0 {
		return message.Content
	}
	return blocks
}

func anthropicBlockFor(part ContentPart) (anthropicBlock, bool) {
	switch part.Type {
	case ContentPartText:
		return anthropicBlock{Type: "text", Text: part.Text}, true
	case ContentPartImageURL:
		if part.ImageURL == nil {
			return anthropicBlock{}, false
		}
		return anthropicMediaBlock("image", part.ImageURL.URL)
	case ContentPartFile:
		if part.File == nil {
			return anthropicBlock{}, false
		}
		if strings.TrimSpace(part.File.FileID) != "" {
			return anthropicBlock{}, false
		}
		return anthropicDocumentBlock(part.File.FileData)
	default:
		return anthropicBlock{}, false
	}
}

func anthropicMediaBlock(kind, value string) (anthropicBlock, bool) {
	if mediaType, data, ok := SplitDataURL(value); ok && strings.HasPrefix(mediaType, "image/") {
		return anthropicBlock{Type: kind, Source: &anthropicSource{Type: "base64", MediaType: mediaType, Data: data}}, true
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return anthropicBlock{Type: kind, Source: &anthropicSource{Type: "url", URL: value}}, true
	}
	return anthropicBlock{}, false
}

func anthropicDocumentBlock(value string) (anthropicBlock, bool) {
	mediaType, data, ok := SplitDataURL(value)
	if !ok {
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return anthropicBlock{Type: "document", Source: &anthropicSource{Type: "url", URL: value}}, true
		}
		return anthropicBlock{}, false
	}
	switch {
	case mediaType == "application/pdf":
		return anthropicBlock{Type: "document", Source: &anthropicSource{Type: "base64", MediaType: mediaType, Data: data}}, true
	case strings.HasPrefix(mediaType, "image/"):
		return anthropicBlock{Type: "image", Source: &anthropicSource{Type: "base64", MediaType: mediaType, Data: data}}, true
	case strings.HasPrefix(mediaType, "text/"):
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return anthropicBlock{}, false
		}
		return anthropicBlock{Type: "document", Source: &anthropicSource{Type: "text", MediaType: mediaType, Data: string(decoded)}}, true
	default:
		return anthropicBlock{}, false
	}
}
