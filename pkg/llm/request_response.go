package llm

import (
	"github.com/praveen001/uno/pkg/llm/embeddings"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type Request struct {
	OfEmbeddingsInput *embeddings.Request
	OfResponsesInput  *responses.Request
}

func (r *Request) GetRequestedModel() string {
	if r.OfResponsesInput != nil {
		return r.OfResponsesInput.Model
	}

	if r.OfEmbeddingsInput != nil {
		return r.OfEmbeddingsInput.Model
	}

	return ""
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
