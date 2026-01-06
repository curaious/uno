package openai_responses

import (
	"github.com/curaious/uno/pkg/llm/responses"
)

func NativeRequestToRequest(in *responses.Request) *Request {
	return &Request{
		*in,
	}
}

func NativeResponseToResponse(in *responses.Response) *Response {
	return &Response{
		*in,
	}
}
