package provider

import (
	"encoding/base64"
	"encoding/json"
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
	Type      string           `json:"type"`
	Text      string           `json:"text,omitempty"`
	Source    *anthropicSource `json:"source,omitempty"`
	ID        string           `json:"id,omitempty"`
	Name      string           `json:"name,omitempty"`
	Input     json.RawMessage  `json:"input,omitempty"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   any              `json:"content,omitempty"`
}

// anthropicMessages splits system prompts out of the message list and lowers
// OpenAI-style content parts, tool calls and tool results into Anthropic
// messages. Media the Anthropic message API cannot carry is replaced by a text
// note and logged.
func anthropicMessages(messages []ChatMessage, system *[]string) []anthropicMessage {
	converted := make([]anthropicMessage, 0, len(messages))
	for _, message := range messages {
		switch message.Role {
		case "system":
			*system = append(*system, message.Content)
			if message.HasMedia() {
				slog.Warn("anthropic system media dropped", "parts", len(message.MediaParts()))
			}
		case "tool":
			block := anthropicBlock{Type: "tool_result", ToolUseID: message.ToolCallID, Content: message.Content}
			if len(converted) > 0 && converted[len(converted)-1].Role == "user" {
				if blocks, ok := converted[len(converted)-1].Content.([]anthropicBlock); ok {
					converted[len(converted)-1].Content = append(blocks, block)
					continue
				}
			}
			converted = append(converted, anthropicMessage{Role: "user", Content: []anthropicBlock{block}})
		default:
			converted = append(converted, anthropicMessage{Role: message.Role, Content: anthropicContent(message)})
		}
	}
	return converted
}

func anthropicContent(message ChatMessage) any {
	if message.Parts == nil && len(message.ToolCalls) == 0 {
		return message.Content
	}
	blocks := make([]anthropicBlock, 0, len(message.Parts)+len(message.ToolCalls))
	degraded := make([]string, 0)
	if message.Parts != nil {
		for _, part := range message.Parts {
			block, ok := anthropicBlockFor(part)
			if !ok {
				degraded = append(degraded, part.Modality())
				blocks = append(blocks, anthropicBlock{Type: "text", Text: omissionNote(part, part.Modality())})
				continue
			}
			blocks = append(blocks, block)
		}
	} else if strings.TrimSpace(message.Content) != "" {
		blocks = append(blocks, anthropicBlock{Type: "text", Text: message.Content})
	}
	if len(degraded) > 0 {
		slog.Warn("anthropic media degraded", "modalities", degraded)
	}
	for _, call := range message.ToolCalls {
		blocks = append(blocks, anthropicBlock{
			Type: "tool_use", ID: call.ID, Name: call.Function.Name, Input: call.Arguments(),
		})
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
