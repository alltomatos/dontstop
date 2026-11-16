package provider

import (
	"fmt"
	"strings"
)

// LimitToModalities returns a copy of the request with unsupported media parts
// replaced by text notes, plus the dropped modalities.
func (r ChatRequest) LimitToModalities(supported Modalities) (ChatRequest, []string) {
	dropped := make([]string, 0)
	messages := make([]ChatMessage, len(r.Messages))
	for index, message := range r.Messages {
		adjusted, partDropped := message.LimitToModalities(supported)
		messages[index] = adjusted
		dropped = append(dropped, partDropped...)
	}
	if len(dropped) == 0 {
		return r, nil
	}
	r.Messages = messages
	return r, dropped
}

// HasMedia reports whether the message carries any non-text content part.
func (m ChatMessage) HasMedia() bool {
	for _, part := range m.Parts {
		if part.Modality() != ModalityText {
			return true
		}
	}
	return false
}

// MediaParts returns the non-text content parts.
func (m ChatMessage) MediaParts() []ContentPart {
	if m.Parts == nil {
		return nil
	}
	media := make([]ContentPart, 0, len(m.Parts))
	for _, part := range m.Parts {
		if part.Modality() != ModalityText {
			media = append(media, part)
		}
	}
	return media
}

// LimitToModalities replaces unsupported media parts with a text note so the
// request can still be served by a text-only model. It returns the adjusted
// message plus the dropped modalities (may repeat).
func (m ChatMessage) LimitToModalities(supported Modalities) (ChatMessage, []string) {
	if m.Parts == nil {
		return m, nil
	}
	dropped := make([]string, 0)
	parts := make([]ContentPart, 0, len(m.Parts))
	for _, part := range m.Parts {
		modality := part.Modality()
		if supported.Supports(modality) {
			parts = append(parts, part)
			continue
		}
		dropped = append(dropped, modality)
		parts = append(parts, ContentPart{Type: ContentPartText, Text: omissionNote(part, modality)})
	}
	if len(dropped) == 0 {
		return m, nil
	}
	m.Parts = parts
	m.Content = joinTextParts(parts)
	return m, dropped
}

func omissionNote(part ContentPart, modality string) string {
	name := modality
	if part.Type == ContentPartFile && part.File != nil && strings.TrimSpace(part.File.Filename) != "" {
		name = part.File.Filename
	}
	return fmt.Sprintf("[attachment omitted: model does not support %s input (%s)]", modality, name)
}

func joinTextParts(parts []ContentPart) string {
	var builder strings.Builder
	for _, part := range parts {
		if part.Type != ContentPartText {
			continue
		}
		if builder.Len() > 0 && part.Text != "" {
			builder.WriteString("\n")
		}
		builder.WriteString(part.Text)
	}
	return builder.String()
}
