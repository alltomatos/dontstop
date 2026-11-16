package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToolFunction describes the JSON schema of a callable function.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// Tool is one callable function exposed to the model.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolCallFunction is the function payload of a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall is a completed tool call emitted by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallDelta is one streaming fragment of a tool call.
type ToolCallDelta struct {
	Index    int             `json:"index"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Function ToolCallDeltaFn `json:"function"`
}

// ToolCallDeltaFn is the function fragment carried by a streaming tool call.
type ToolCallDeltaFn struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Validate rejects tools that would be forwarded to a provider in an invalid shape.
func (t Tool) Validate() error {
	if strings.TrimSpace(t.Type) != "function" {
		return fmt.Errorf("tool type must be function")
	}
	if strings.TrimSpace(t.Function.Name) == "" {
		return fmt.Errorf("tool function name is required")
	}
	if len(t.Function.Parameters) > 0 && !json.Valid(t.Function.Parameters) {
		return fmt.Errorf("tool %q parameters must be valid JSON", t.Function.Name)
	}
	return nil
}

// ValidateToolChoice accepts the OpenAI union: auto, none, required or a
// function selector object.
func ValidateToolChoice(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("tool_choice must be valid JSON")
	}
	switch typed := value.(type) {
	case string:
		switch typed {
		case "auto", "none", "required":
			return nil
		default:
			return fmt.Errorf("tool_choice must be auto, none or required")
		}
	case map[string]any:
		function, ok := typed["function"].(map[string]any)
		if !ok {
			return fmt.Errorf("tool_choice.function is required")
		}
		if name, ok := function["name"].(string); !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("tool_choice.function.name is required")
		}
		return nil
	default:
		return fmt.Errorf("tool_choice must be a string or a function selector")
	}
}

// ToolChoiceName returns the forced function name when tool_choice selects one.
func ToolChoiceName(raw json.RawMessage) string {
	var value struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value.Function.Name)
}

// ToolChoiceMode returns auto, none or required when tool_choice is a string.
func ToolChoiceMode(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

// ToolCallArguments parses the JSON arguments of a tool call.
func (c ToolCall) Arguments() json.RawMessage {
	arguments := []byte(c.Function.Arguments)
	if len(arguments) == 0 || !json.Valid(arguments) {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(arguments)
}
