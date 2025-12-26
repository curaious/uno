package openai_embeddings

import "github.com/praveen001/uno/pkg/llm/embeddings"

type Request struct {
	embeddings.Request
}

func (r *Request) ToNativeRequest() *embeddings.Request {
	return &r.Request
}

func NativeRequestToRequest(in *embeddings.Request) *Request {
	return &Request{*in}
}

type Response struct {
	embeddings.Response
}

func (r *Response) ToNativeResponse() *embeddings.Response {
	return &r.Response
}

func NativeResponseToResponse(in *embeddings.Response) *Response {
	return &Response{*in}
}
