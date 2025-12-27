package logger

import (
	"context"
	"log/slog"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
)

type LoggerMiddleware struct {
}

func NewLoggerMiddleware() *LoggerMiddleware {
	return &LoggerMiddleware{}
}

func (middleware *LoggerMiddleware) HandleRequest(next gateway.RequestHandler) gateway.RequestHandler {
	return func(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.Response, error) {
		return next(ctx, providerName, key, r)
	}
}

func (middleware *LoggerMiddleware) HandleStreamingRequest(next gateway.StreamingRequestHandler) gateway.StreamingRequestHandler {
	return func(ctx context.Context, providerName llm.ProviderName, key string, r *llm.Request) (*llm.StreamingResponse, error) {
		slog.InfoContext(ctx, "Start processing streaming request", slog.String("provider", string(providerName)))
		res, err := next(ctx, providerName, key, r)
		if err != nil {
			slog.ErrorContext(ctx, "Error processing streaming request", slog.String("provider", string(providerName)), slog.Any("error", err))
		}
		slog.InfoContext(ctx, "Finished processing streaming request", slog.String("provider", string(providerName)))
		return res, err
	}
}
