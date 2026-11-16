package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Content part types accepted by the OpenAI-compatible surface.
const (
	ContentPartText       = "text"
	ContentPartImageURL   = "image_url"
	ContentPartInputAudio = "input_audio"
	ContentPartFile       = "file"
)

// Input modality names.
const (
	ModalityText  = "text"
	ModalityImage = "image"
	ModalityAudio = "audio"
	ModalityVideo = "video"
	ModalityPDF   = "pdf"
	ModalityFile  = "file"
)

// ImageURL is the image_url payload of an image content part.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// InputAudio is the input_audio payload of an audio content part.
type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

// FileData is the file payload of a document content part.
type FileData struct {
	Filename string `json:"filename,omitempty"`
	FileData string `json:"file_data,omitempty"`
	FileID   string `json:"file_id,omitempty"`
}

// ContentPart is one element of a multimodal message.
type ContentPart struct {
	Type       string      `json:"type"`
	Text       string      `json:"text,omitempty"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
	File       *FileData   `json:"file,omitempty"`
}

// Validate rejects malformed content parts before they reach a provider.
func (p ContentPart) Validate() error {
	switch p.Type {
	case ContentPartText:
		return nil
	case ContentPartImageURL:
		if p.ImageURL == nil || strings.TrimSpace(p.ImageURL.URL) == "" {
			return fmt.Errorf("image_url part requires a url")
		}
		return nil
	case ContentPartInputAudio:
		if p.InputAudio == nil || strings.TrimSpace(p.InputAudio.Data) == "" {
			return fmt.Errorf("input_audio part requires data")
		}
		switch strings.ToLower(p.InputAudio.Format) {
		case "wav", "mp3":
			return nil
		default:
			return fmt.Errorf("input_audio part requires format wav or mp3")
		}
	case ContentPartFile:
		if p.File == nil || (strings.TrimSpace(p.File.FileData) == "" && strings.TrimSpace(p.File.FileID) == "") {
			return fmt.Errorf("file part requires file_data or file_id")
		}
		return nil
	default:
		return fmt.Errorf("unsupported content part type %q", p.Type)
	}
}

// Modality reports the input modality required by the content part.
func (p ContentPart) Modality() string {
	switch p.Type {
	case ContentPartText:
		return ModalityText
	case ContentPartImageURL:
		return ModalityImage
	case ContentPartInputAudio:
		return ModalityAudio
	case ContentPartFile:
		return fileModality(p.File)
	default:
		return ModalityFile
	}
}

func fileModality(file *FileData) string {
	if file == nil {
		return ModalityFile
	}
	mediaType, _, ok := SplitDataURL(file.FileData)
	if !ok {
		return ModalityFile
	}
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		return ModalityImage
	case strings.HasPrefix(mediaType, "audio/"):
		return ModalityAudio
	case strings.HasPrefix(mediaType, "video/"):
		return ModalityVideo
	case mediaType == "application/pdf":
		return ModalityPDF
	default:
		return ModalityFile
	}
}

// SplitDataURL splits a data: URL into media type and raw payload.
func SplitDataURL(value string) (mediaType, data string, ok bool) {
	if !strings.HasPrefix(value, "data:") {
		return "", "", false
	}
	rest := strings.TrimPrefix(value, "data:")
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return "", "", false
	}
	header := rest[:comma]
	data = rest[comma+1:]
	parts := strings.Split(header, ";")
	if len(parts) < 2 || !strings.EqualFold(parts[len(parts)-1], "base64") {
		return "", "", false
	}
	return parts[0], data, true
}

// Modalities describes the input modalities a model accepts.
type Modalities struct {
	Text  bool
	Image bool
	Audio bool
	Video bool
	PDF   bool
}

// Supports reports whether the model accepts one input modality.
func (m Modalities) Supports(modality string) bool {
	switch modality {
	case ModalityText:
		return m.Text
	case ModalityImage:
		return m.Image
	case ModalityAudio:
		return m.Audio
	case ModalityVideo:
		return m.Video
	case ModalityPDF:
		return m.PDF
	default:
		return false
	}
}

// Inputs returns the supported input modalities in a stable order.
func (m Modalities) Inputs() []string {
	inputs := make([]string, 0, 5)
	for _, entry := range []struct {
		enabled bool
		name    string
	}{
		{m.Text, ModalityText},
		{m.Image, ModalityImage},
		{m.Audio, ModalityAudio},
		{m.Video, ModalityVideo},
		{m.PDF, ModalityPDF},
	} {
		if entry.enabled {
			inputs = append(inputs, entry.name)
		}
	}
	return inputs
}

// ParseModalities reads the capabilities JSONB stored on provider_models.
// It returns ok=false when the document carries no modality keys.
func ParseModalities(raw []byte) (Modalities, bool) {
	if len(raw) == 0 {
		return Modalities{}, false
	}
	var caps map[string]any
	if err := json.Unmarshal(raw, &caps); err != nil {
		return Modalities{}, false
	}
	enabled := func(keys ...string) bool {
		for _, key := range keys {
			if value, ok := caps[key].(bool); ok {
				return value
			}
		}
		return false
	}
	found := false
	for _, key := range []string{"text_input", "image_input", "audio_input", "video_input", "pdf_input", "vision", "audio"} {
		if _, ok := caps[key]; ok {
			found = true
			break
		}
	}
	if !found {
		return Modalities{}, false
	}
	return Modalities{
		Text:  enabled("text_input"),
		Image: enabled("image_input", "vision"),
		Audio: enabled("audio_input", "audio"),
		Video: enabled("video_input"),
		// The catalog tracks document support through vision today.
		PDF: enabled("pdf_input", "image_input", "vision"),
	}, true
}
