package anthropic_responses

import (
	"errors"

	"github.com/bytedance/sonic"
)

type Request struct {
	MaxTokens int            `json:"max_tokens"`
	Model     string         `json:"model"`
	Messages  []MessageUnion `json:"messages"`

	System       []TextContent     `json:"system,omitempty"`
	Temperature  *float64          `json:"temperature,omitempty"`
	TopK         *int64            `json:"top_k,omitempty"`
	TopP         *float64          `json:"top_p,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Thinking     *ThinkingParam    `json:"thinking,omitempty"`
	Tools        []ToolUnion       `json:"tools,omitempty"`
	Stream       *bool             `json:"stream,omitempty"`
	OutputFormat map[string]any    `json:"output_format,omitempty"`
}

type ThinkingParam struct {
	Type         *string `json:"type"` // "enabled" or "disabled"
	BudgetTokens *int    `json:"budget_tokens"`
}

type MessageUnion struct {
	Role    Role     `json:"role"` // "user" or "assistant"
	Content Contents `json:"content"`
}

type Contents []ContentUnion

type ContentUnion struct {
	OfText             *TextContent             `json:",omitempty"`
	OfToolUse          *ToolUseContent          `json:",omitempty"`
	OfToolResult       *ToolUseResultContent    `json:",omitempty"`
	OfThinking         *ThinkingContent         `json:",omitempty"`
	OfRedactedThinking *RedactedThinkingContent `json:",omitempty"`
}

func (u *ContentUnion) UnmarshalJSON(data []byte) error {
	var textContext TextContent
	if err := sonic.Unmarshal(data, &textContext); err == nil {
		u.OfText = &textContext
		return nil
	}

	var toolUseContent ToolUseContent
	if err := sonic.Unmarshal(data, &toolUseContent); err == nil {
		u.OfToolUse = &toolUseContent
		return nil
	}

	var toolUseResultContent ToolUseResultContent
	if err := sonic.Unmarshal(data, &toolUseResultContent); err == nil {
		u.OfToolResult = &toolUseResultContent
		return nil
	}

	var thinkingContent ThinkingContent
	if err := sonic.Unmarshal(data, &thinkingContent); err == nil {
		u.OfThinking = &thinkingContent
		return nil
	}

	var redactedThinkingContent RedactedThinkingContent
	if err := sonic.Unmarshal(data, &redactedThinkingContent); err == nil {
		u.OfRedactedThinking = &redactedThinkingContent
		return nil
	}

	return errors.New("invalid input content union")
}

func (u *ContentUnion) MarshalJSON() ([]byte, error) {
	if u.OfText != nil {
		return sonic.Marshal(*u.OfText)
	}

	if u.OfToolUse != nil {
		return sonic.Marshal(*u.OfToolUse)
	}

	if u.OfToolResult != nil {
		return sonic.Marshal(*u.OfToolResult)
	}

	if u.OfThinking != nil {
		return sonic.Marshal(*u.OfThinking)
	}

	if u.OfRedactedThinking != nil {
		return sonic.Marshal(*u.OfRedactedThinking)
	}

	return nil, nil
}

type TextContent struct {
	Type ContentTypeText `json:"type"` // "text"
	Text string          `json:"text"`
}

type ToolUseContent struct {
	Type  ContentTypeToolUse `json:"type"`
	ID    string             `json:"id"`
	Name  string             `json:"name"`
	Input any                `json:"input"`
}

type ToolUseResultContent struct {
	Type      ContentTypeToolUseResult `json:"type"` // "tool_result"
	ToolUseID string                   `json:"tool_use_id"`
	Content   []ContentUnion           `json:"content"`
	IsError   *bool                    `json:"is_error,omitempty"`
}

type ThinkingContent struct {
	Type      ContentTypeThinking `json:"type"`
	Thinking  string              `json:"thinking"`
	Signature string              `json:"signature"`
}

type RedactedThinkingContent struct {
	Type ContentTypeRedactedThinking `json:"type"`
	Data string                      `json:"data"`
}

type ToolUnion struct {
	OfCustomTool *CustomTool `json:",omitempty"`
}

func (u *ToolUnion) UnmarshalJSON(data []byte) error {
	var customTool CustomTool
	if err := sonic.Unmarshal(data, &customTool); err == nil {
		u.OfCustomTool = &customTool
		return nil
	}

	return errors.New("invalid tool union")
}

func (u *ToolUnion) MarshalJSON() ([]byte, error) {
	if u.OfCustomTool != nil {
		return sonic.Marshal(u.OfCustomTool)
	}

	return nil, nil
}

type CustomTool struct {
	Type        string         `json:"type"` // "custom"
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}
