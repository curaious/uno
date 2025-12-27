package chat_completion

import "github.com/praveen001/uno/pkg/llm/constants"

type Request struct {
	Messages            ChatCompletionMessageUnion `json:"messages"`
	Model               string                     `json:"model"`
	FrequencyPenalty    *float64                   `json:"frequency_penalty,omitempty"`
	Logprobs            *bool                      `json:"logprobs,omitempty"`
	MaxCompletionTokens *int64                     `json:"max_completion_tokens,omitempty"`
	MaxTokens           *int64                     `json:"max_tokens,omitempty"` // Deprecated in favour of `MaxCompletionTokens`
	N                   *int64                     `json:"n,omitempty"`          // How many choices needs to be generated
	PresencePenalty     *float64                   `json:"presence_penalty,omitempty"`
	Seed                *int64                     `json:"seed,omitempty"`
	Store               *bool                      `json:"store,omitempty"`
	Temperature         *float64                   `json:"temperature,omitempty"`
	TopLogprobs         *int64                     `json:"top_logprobs,omitempty"`
	TopP                *float64                   `json:"top_p,omitempty"`
	ParallelToolCalls   *bool                      `json:"parallel_tool_calls,omitempty"`
	PromptCacheKey      *string                    `json:"prompt_cache_key,omitempty"`
	SafetyIdentifier    *string                    `json:"safety_identifier,omitempty"`
	User                *string                    `json:"user,omitempty"`
	Audio               *AudioParam                `json:"audio,omitempty"`
	LogitBias           map[string]int64           `json:"logit_bias,omitempty"`
	Metadata            map[string]string          `json:"metadata,omitempty"`
	Modalities          []string                   `json:"modalities,omitempty"`       // "text", "audio"
	ReasoningEffort     *string                    `json:"reasoning_effort,omitempty"` // "minimal", "low", "medium", "high"
	ServiceTier         *string                    `json:"service_tier,omitempty"`     // "auto", "default", "flex", "scale", "priority"
	Stop                *StopParam                 `json:"stop,omitempty"`
	StreamOptions       *StreamOptionParam         `json:"stream_options,omitempty"` // Set only when setting stream=true
	Verbosity           *string                    `json:"verbosity,omitempty"`      // "low", "medium", "high"
	FunctionCall        *FunctionCallParam         `json:"function_call,omitempty"`  // Deprecated in favour of `tool_choice`
	Functions           []FunctionsParam           `json:"functions,omitempty"`      // Deprecated in favour of tools
	Prediction          any                        `json:"prediction,omitempty"`
	ResponseFormat      any                        `json:"response_format,omitempty"`
	ToolChoice          *string                    `json:"tool_choice,omitempty"`
	Tools               any                        `json:"tools,omitempty"`
	WebSearchOptions    any                        `json:"web_search_options,omitempty"`
}

type AudioParam struct {
	Voice  string `json:"voice"`
	Format string `json:"format"`
}

type StopParam struct {
	OfString *string  `json:",omitempty"`
	OfList   []string `json:",omitempty"`
}

type StreamOptionParam struct {
	IncludeObfuscation *bool `json:"include_obfuscation,omitempty"`
	IncludeUsage       *bool `json:"include_usage,omitempty"`
}

type FunctionCallParam struct {
	OfFunctionCallMode   *string                  `json:",omitempty"`
	OfFunctionCallOption *FunctionCallOptionParam `json:",omitempty"`
}

type FunctionCallOptionParam struct {
	Name string `json:"name"`
}

type FunctionsParam struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ChatCompletionMessageUnion struct {
	OfDeveloper *DeveloperChatCompletionMessageUnion `json:",omitzero"`
	OfSystem    *SystemChatCompletionMessageUnion    `json:",omitzero"`
	OfUser      *UserChatCompletionMessageUnion      `json:",omitzero"`
	OfAssistant *AssistantChatCompletionMessageUnion `json:",omitzero"`
	OfTool      *ToolChatCompletionMessageUnion      `json:",omitzero"`
	OfFunction  *FunctionChatCompletionMessageUnion  `json:",omitzero"`
}

type DeveloperChatCompletionMessageUnion struct {
	Name    *string                      `json:"name,omitempty"`
	Role    constants.Role               `json:"role,omitempty"` // "developer"
	Content DeveloperMessageContentUnion `json:"content,omitempty"`
}

type SystemChatCompletionMessageUnion struct {
	Name    *string                   `json:"name,omitempty"`
	Role    constants.Role            `json:"role,omitempty"` // system
	Content SystemMessageContentUnion `json:"content,omitempty"`
}

type UserChatCompletionMessageUnion struct {
	Name    *string                 `json:"name,omitempty"`
	Role    constants.Role          `json:"role,omitempty"` // user
	Content UserMessageContentUnion `json:"content,omitempty"`
}

type AssistantChatCompletionMessageUnion struct {
	Refusal      *string                         `json:"refusal,omitempty"`
	Name         *string                         `json:"name,omitempty"`
	Audio        AssistantMessageAudio           `json:"audio,omitempty"`
	Content      AssistantMessageContentUnion    `json:"content,omitempty"`
	FunctionCall AssistantMessageFunctionCall    `json:"function_call,omitempty"`
	ToolCalls    []AssistantMessageToolCallUnion `json:"tool_calls,omitempty"`
	Role         *string                         `json:"role,omitempty"` // "assistant"
}
type ToolChatCompletionMessageUnion struct {
	Role       constants.Role          `json:"role,omitempty"` // tool
	Content    ToolMessageContentUnion `json:"content,omitempty"`
	ToolCallID string                  `json:"tool_call_id,omitempty"`
}
type FunctionChatCompletionMessageUnion struct {
	Name    *string        `json:"name,omitempty"`
	Role    constants.Role `json:"role,omitempty"` //
	Content *string        `json:"content,omitempty"`
}

type DeveloperMessageContentUnion struct {
	OfString *string    `json:",omitempty"`
	OfList   []TextPart `json:",omitempty"`
}

type SystemMessageContentUnion struct {
	OfString *string    `json:",omitempty"`
	OfList   []TextPart `json:",omitempty"`
}

type UserMessageContentUnion struct {
	OfString *string                       `json:",omitempty"`
	OfList   []UserMessageContentPartUnion `json:",omitempty"`
}

type AssistantMessageAudio struct {
	ID string `json:"id"`
}

type AssistantMessageContentUnion struct {
	OfString *string                            `json:",omitempty"`
	OfList   []AssistantMessageContentPartUnion `json:",omitempty"`
}

type AssistantMessageFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

type AssistantMessageToolCallUnion struct {
	OfFunction AssistantMessageFunctionToolCall `json:",omitempty"`
	OfCustom   AssistantMessageCustomToolCall   `json:",omitempty"`
}

type AssistantMessageFunctionToolCall struct {
	Type     string                                `json:"type"` // "function"
	ID       string                                `json:"id,omitempty"`
	Function AssistantMessageFunctionToolCallParam `json:"function"`
}

type AssistantMessageFunctionToolCallParam struct {
	Name      string `json:"name"`
	Arguments string `json:"input"`
}

type AssistantMessageCustomToolCall struct {
	Type   string                              `json:"type"` // "custom"
	ID     string                              `json:"id,omitempty"`
	Custom AssistantMessageCustomToolCallParam `json:"custom"`
}

type AssistantMessageCustomToolCallParam struct {
	Name  string `json:"name"`
	Input string `json:"input"`
}

type ToolMessageContentUnion struct {
	OfString *string    `json:",omitempty"`
	OfList   []TextPart `json:",omitempty"`
}

type AssistantMessageContentPartUnion struct {
	OfText    *TextPart    `json:",omitempty"`
	OfRefusal *RefusalPart `json:",omitempty"`
}

type UserMessageContentPartUnion struct {
	OfText       *TextPart  `json:",omitempty"`
	OfImageUrl   *ImagePart `json:",omitempty"`
	OfInputAudio *AudioPart `json:",omitempty"`
	OfFile       *FilePart  `json:",omitempty"`
}

type RefusalPart struct {
	Type    string `json:"type"` // "refusal"
	Refusal string `json:"refusal,omitempty"`
}

type TextPart struct {
	Type string `json:"type,omitempty"` // "text"
	Text string `json:"text,omitempty"`
}

type ImagePart struct {
	Type     string   `json:"type,omitempty"` // "image_url"
	ImageUrl ImageUrl `json:"image_url,omitempty"`
}

type AudioPart struct {
	Type       string     `json:"type,omitempty"` // "input_audio"
	InputAudio InputAudio `json:"input_audio,omitempty"`
}

type FilePart struct {
	Type string `json:"type,omitempty"`
	File File   `json:"file,omitempty"`
}

type ImageUrl struct {
	Url    string `json:"url"`
	Detail string `json:"detail"` // "auto", "low", "high"
}

type InputAudio struct {
	Format string `json:"format,omitzero"` // "wav", "mp3"
	Data   string `json:"data"`
}

type File struct {
	FileID   *string `json:"file_id,omitempty"`
	Filename *string `json:"filename,omitempty"`
	FileData *string `json:"file_data,omitempty"`
}
