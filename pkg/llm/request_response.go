package llm

import (
	"github.com/praveen001/uno/pkg/llm/embeddings"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type Request struct {
	OfEmbeddingsInput *embeddings.Request
	OfResponsesInput  *responses.Request
}

type Response struct {
	OfEmbeddingsOutput *embeddings.Response
	OfResponsesOutput  *responses.Response
	Error              *Error
}

type StreamingResponse struct {
	ResponsesStreamData chan *responses.ResponseChunk
}

type Error struct {
	Message string
}
