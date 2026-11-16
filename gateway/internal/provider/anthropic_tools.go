package provider

import (
	"encoding/json"
)

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// anthropicTools lowers the OpenAI tool surface into Anthropic tools and
// tool_choice. A "none" choice omits tools entirely because Anthropic has no
// equivalent none mode.
func anthropicTools(request ChatRequest) ([]anthropicTool, *anthropicChoice) {
	if len(request.Tools) == 0 {
		return nil, nil
	}
	mode := ToolChoiceMode(request.ToolChoice)
	if mode == "none" {
		return nil, nil
	}
	tools := make([]anthropicTool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		schema := tool.Function.Parameters
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		tools = append(tools, anthropicTool{
			Name: tool.Function.Name, Description: tool.Function.Description, InputSchema: schema,
		})
	}
	switch mode {
	case "auto":
		return tools, &anthropicChoice{Type: "auto"}
	case "required":
		return tools, &anthropicChoice{Type: "any"}
	}
	if name := ToolChoiceName(request.ToolChoice); name != "" {
		return tools, &anthropicChoice{Type: "tool", Name: name}
	}
	return tools, nil
}
