package openai_responses

import (
	"github.com/praveen001/uno/pkg/llm/responses"
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
