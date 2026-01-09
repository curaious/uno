package llm

import (
	"github.com/curaious/uno/pkg/llm/chat_completion"
	"github.com/curaious/uno/pkg/llm/embeddings"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/curaious/uno/pkg/llm/speech"
)

type Request struct {
	OfEmbeddingsInput     *embeddings.Request
	OfResponsesInput      *responses.Request
	OfChatCompletionInput *chat_completion.Request
	OfSpeech              *speech.Request
}

func (r *Request) GetRequestedModel() string {
	if r.OfResponsesInput != nil {
		return r.OfResponsesInput.Model
	}

	if r.OfEmbeddingsInput != nil {
		return r.OfEmbeddingsInput.Model
	}

	if r.OfChatCompletionInput != nil {
		return r.OfChatCompletionInput.Model
	}

	if r.OfSpeech != nil {
		return r.OfSpeech.Model
	}

	return ""
}

type Response struct {
	OfEmbeddingsOutput     *embeddings.Response
	OfResponsesOutput      *responses.Response
	OfChatCompletionOutput *chat_completion.Response
	OfSpeech               *speech.Response
	Error                  *Error
}

type StreamingResponse struct {
	ResponsesStreamData      chan *responses.ResponseChunk
	ChatCompletionStreamData chan *chat_completion.ResponseChunk
	SpeechStreamData         chan *speech.ResponseChunk
}

type Error struct {
	Message string
}
