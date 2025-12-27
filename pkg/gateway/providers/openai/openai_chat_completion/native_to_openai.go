package openai_chat_completion

import (
	"github.com/praveen001/uno/pkg/llm/chat_completion"
)

func NativeRequestToRequest(in *chat_completion.Request) *Request {
	return &Request{
		*in,
	}
}

func NativeResponseToResponse(in *chat_completion.Response) *Response {
	return &Response{
		*in,
	}
}
