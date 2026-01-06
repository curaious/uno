package openai_chat_completion

import (
	"github.com/curaious/uno/pkg/llm/chat_completion"
)

func (in *Request) ToNativeRequest() *chat_completion.Request {
	return &in.Request
}

func (in *Response) ToNativeResponse() *chat_completion.Response {
	return &in.Response
}
