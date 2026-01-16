package temporal_runtime

import (
	"context"
	"log/slog"

	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type TemporalLLM struct {
	wrappedLLM llm.Provider
	broker     core.StreamBroker
}

func NewTemporalLLM(wrappedLLM llm.Provider, broker core.StreamBroker) *TemporalLLM {
	return &TemporalLLM{
		wrappedLLM: wrappedLLM,
		broker:     broker,
	}
}

func (l *TemporalLLM) NewStreamingResponsesActivity(ctx context.Context, in *responses.Request) (*responses.Response, error) {
	stream, err := l.wrappedLLM.NewStreamingResponses(ctx, in)
	if err != nil {
		return nil, err
	}

	acc := agents.Accumulator{}
	resp, err := acc.ReadStream(stream, func(chunk *responses.ResponseChunk) {
		if err := l.broker.Publish(ctx, activity.GetInfo(ctx).WorkflowExecution.ID, chunk); err != nil {
			slog.ErrorContext(ctx, "Failed to publish chunk to stream broker", "error", err)
		}
	})
	l.broker.Close(ctx, activity.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type TemporalLLMProxy struct {
	workflowCtx workflow.Context
	prefix      string
	broker      core.StreamBroker
}

func NewTemporalLLMProxy(workflowCtx workflow.Context, prefix string, broker core.StreamBroker) agents.LLM {
	return &TemporalLLMProxy{
		workflowCtx: workflowCtx,
		prefix:      prefix,
		broker:      broker,
	}
}

func (l *TemporalLLMProxy) NewStreamingResponses(ctx context.Context, in *responses.Request, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error) {
	var response *responses.Response
	err := workflow.ExecuteActivity(l.workflowCtx, l.prefix+"_NewStreamingResponsesActivity", in).Get(l.workflowCtx, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
