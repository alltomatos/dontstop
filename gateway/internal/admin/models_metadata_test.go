package admin

import "testing"

func TestRemoteModelMetadataParsesOpenCodePayload(t *testing.T) {
	model := RemoteModel{
		ID:   "kimi-k3",
		Type: "chat",
		Modalities: struct {
			Input  []string `json:"input"`
			Output []string `json:"output"`
		}{
			Input:  []string{"text", "image/jpeg", "video/mp4"},
			Output: []string{"text"},
		},
		Capabilities: struct {
			Tools struct {
				Supported bool `json:"supported"`
			} `json:"tools"`
			Reasoning struct {
				Supported bool `json:"supported"`
			} `json:"reasoning"`
		}{},
		Limit: struct {
			Context int `json:"context"`
			Output  int `json:"output"`
		}{Context: 1048576, Output: 131072},
		Cost: struct {
			Text struct {
				Input struct {
					Price float64 `json:"price"`
				} `json:"input"`
				Output struct {
					Price float64 `json:"price"`
				} `json:"output"`
			} `json:"text"`
		}{},
	}
	model.Capabilities.Tools.Supported = true
	model.Capabilities.Reasoning.Supported = true
	model.Cost.Text.Input.Price = 3
	model.Cost.Text.Output.Price = 15

	capabilities, inputPrice, outputPrice := remoteModelMetadata(model)
	if capabilities["text_input"] != true || capabilities["image_input"] != true ||
		capabilities["video_input"] != true || capabilities["image_output"] != false {
		t.Fatalf("unexpected modalities: %+v", capabilities)
	}
	if capabilities["tools"] != true || capabilities["reasoning"] != true ||
		capabilities["stream"] != true || capabilities["embeddings"] != false {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if capabilities["context_window"] != 1048576 || capabilities["max_output_tokens"] != 131072 {
		t.Fatalf("unexpected limits: %+v", capabilities)
	}
	if inputPrice != 3 || outputPrice != 15 || capabilities["pricing_known"] != true {
		t.Fatalf("unexpected pricing: input=%v output=%v caps=%+v", inputPrice, outputPrice, capabilities)
	}
	if capabilities["metadata_source"] != "provider:/models" {
		t.Fatalf("unexpected metadata source: %+v", capabilities)
	}
}

func TestRemoteModelMetadataDoesNotInferFromName(t *testing.T) {
	model := RemoteModel{ID: "vision-pro-max"}
	capabilities, _, _ := remoteModelMetadata(model)
	if capabilities["image_input"] != nil || capabilities["vision"] != nil || capabilities["tools"] != nil {
		t.Fatalf("capabilities must not be inferred from the model name: %+v", capabilities)
	}
	if capabilities["metadata_source"] != "" {
		t.Fatalf("metadata must be empty: %+v", capabilities)
	}
}
