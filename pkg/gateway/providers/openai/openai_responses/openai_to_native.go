package openai_responses

import (
	"github.com/curaious/uno/pkg/llm/responses"
)

func (in *Request) ToNativeRequest() *responses.Request {
	return &responses.Request{
		Model:        in.Model,
		Input:        in.Input,
		Instructions: in.Instructions,
		Tools:        in.Tools,
		Parameters: responses.Parameters{
			Background:        in.Background,
			MaxOutputTokens:   in.MaxOutputTokens,
			MaxToolCalls:      in.MaxToolCalls,
			ParallelToolCalls: in.ParallelToolCalls,
			Store:             in.Store,
			Temperature:       in.Temperature,
			TopLogprobs:       in.TopLogprobs,
			TopP:              in.TopP,
			Include:           in.Include,
			Metadata:          in.Metadata,
			Stream:            in.Stream,
			Reasoning:         in.Reasoning,
			Text:              in.Text,
		},
	}
}

func (in *Response) ToNativeResponse() *responses.Response {
	return &in.Response
}

func (in *ResponseChunk) ToNativeResponseChunk() *responses.ResponseChunk {
	return &in.ResponseChunk
}
