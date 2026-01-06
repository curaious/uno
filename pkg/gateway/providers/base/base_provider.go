package base

import (
	"context"

	"github.com/curaious/uno/pkg/llm/chat_completion"
	"github.com/curaious/uno/pkg/llm/embeddings"
	"github.com/curaious/uno/pkg/llm/responses"
)

type BaseProvider struct{}

func (bp *BaseProvider) NewResponses(ctx context.Context, in *responses.Request) (*responses.Response, error) {
	panic("implement me")
}

func (bp *BaseProvider) NewStreamingResponses(ctx context.Context, in *responses.Request) (chan *responses.ResponseChunk, error) {
	panic("implement me")
}

func (bp *BaseProvider) NewEmbedding(ctx context.Context, in *embeddings.Request) (*embeddings.Response, error) {
	panic("implement me")
}

func (bp *BaseProvider) NewChatCompletion(ctx context.Context, in *chat_completion.Request) (*chat_completion.Response, error) {
	panic("implement me")
}

func (bp *BaseProvider) NewStreamingChatCompletion(ctx context.Context, in *chat_completion.Request) (chan *chat_completion.ResponseChunk, error) {
	panic("implement me")
}
