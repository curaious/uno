package xai_responses

import (
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_responses"
	"github.com/curaious/uno/pkg/llm/responses"
)

func NativeRequestToRequest(in *responses.Request) *Request {
	r := &Request{
		Request: &openai_responses.Request{
			*in,
		},
	}

	// Grok doesn't support reasoning effort except for older models like grok-3
	if in.Reasoning != nil {
		r.Reasoning.Effort = nil
	}

	return r
}

func NativeResponseToResponse(in *responses.Response) *Response {
	return &Response{
		*in,
	}
}
