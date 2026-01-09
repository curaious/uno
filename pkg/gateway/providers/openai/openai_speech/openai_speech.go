package openai_speech

import "github.com/curaious/uno/pkg/llm/speech"

type Request struct {
	speech.Request
}

func (r *Request) ToNativeRequest() *speech.Request {
	return &r.Request
}

func NativeRequestToRequest(in *speech.Request) *Request {
	return &Request{*in}
}

type Response struct {
	speech.Response
}

func (r *Response) ToNativeResponse() *speech.Response {
	return &r.Response
}

func NativeResponseToResponse(in *speech.Response) *Response {
	return &Response{*in}
}

type ResponseChunk struct {
	speech.ResponseChunk
}

func (r *ResponseChunk) ToNativeResponse() *speech.ResponseChunk {
	return &r.ResponseChunk
}
