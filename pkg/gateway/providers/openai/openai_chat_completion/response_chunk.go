package openai_chat_completion

import "github.com/curaious/uno/pkg/llm/chat_completion"

type ResponseChunk struct {
	chat_completion.ResponseChunk
}

func (in *ResponseChunk) ToNativeResponseChunk() *chat_completion.ResponseChunk {
	return &in.ResponseChunk
}
