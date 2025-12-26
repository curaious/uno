package xai_responses

import (
	"github.com/praveen001/uno/pkg/gateway/providers/openai/openai_responses"
	"github.com/praveen001/uno/pkg/llm/responses"
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
