package llm

import (
	"github.com/praveen001/uno/pkg/llm/responses"
)

type Request struct {
	OfResponsesInput *responses.Request
}

type Response struct {
	OfResponsesOutput *responses.Response
	Error             *Error
}

type StreamingResponse struct {
	ResponsesStreamData chan *responses.ResponseChunk
}

type Error struct {
	Message string
}
