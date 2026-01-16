package temporal_agent_builder

import (
	"context"
	"log/slog"

	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

func (b *AgentBuilder) LLMCall(ctx context.Context, config *agent_config.ModelConfig, in *responses.Request) (*responses.Response, error) {
	llmClient := builder.BuildLLMClient(b.llmGateway, "", llm.ProviderName(config.ProviderType), config.ModelID)

	stream, err := llmClient.NewStreamingResponses(ctx, in)
	if err != nil {
		return nil, err
	}

	acc := agents.Accumulator{}
	resp, err := acc.ReadStream(stream, func(chunk *responses.ResponseChunk) {
		if err := b.broker.Publish(ctx, activity.GetInfo(ctx).WorkflowExecution.ID, chunk); err != nil {
			slog.ErrorContext(ctx, "Failed to publish chunk to stream broker", "error", err)
		}
	})
	b.broker.Close(ctx, activity.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type TemporalLLMProxy struct {
	workflowCtx workflow.Context
	config      *agent_config.ModelConfig
}

func NewTemporalLLMProxy(workflowCtx workflow.Context, config *agent_config.ModelConfig) agents.LLM {
	return &TemporalLLMProxy{
		workflowCtx: workflowCtx,
		config:      config,
	}
}

func (l *TemporalLLMProxy) NewStreamingResponses(ctx context.Context, in *responses.Request, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error) {
	var response *responses.Response
	err := workflow.ExecuteActivity(l.workflowCtx, "LLMCall", l.config, in).Get(l.workflowCtx, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
