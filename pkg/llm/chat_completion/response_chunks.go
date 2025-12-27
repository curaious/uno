package chat_completion

type ResponseChunk struct {
	OfChatCompletionChunk *ChatCompletionChunk `json:",omitempty"`
}

type ChatCompletionChunk struct {
	Id                string                      `json:"id"`
	Object            string                      `json:"object"`
	Created           int                         `json:"created"`
	Model             string                      `json:"model"`
	ServiceTier       string                      `json:"service_tier"`
	SystemFingerprint interface{}                 `json:"system_fingerprint"`
	Choices           []ChatCompletionChunkChoice `json:"choices"`
	Obfuscation       string                      `json:"obfuscation"`
}

type ChatCompletionChunkChoice struct {
	Index        int                            `json:"index"`
	Delta        ChatCompletionChunkChoiceDelta `json:"delta"`
	FinishReason interface{}                    `json:"finish_reason"`
}

type ChatCompletionChunkChoiceDelta struct {
	Role    string      `json:"role"`
	Content string      `json:"content"`
	Refusal interface{} `json:"refusal"`
}
