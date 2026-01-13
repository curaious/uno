package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type TemporalAgent struct {
	wrappedAgent *agents.Agent
}

// NewTemporalAgent creates a new TemporalAgent.
func NewTemporalAgent(wrappedAgent *agents.Agent) *TemporalAgent {
	return &TemporalAgent{
		wrappedAgent: wrappedAgent,
	}
}

func (a *TemporalAgent) Execute(ctx workflow.Context, in *agents.AgentInput) (*agents.AgentOutput, error) {
	executor := NewTemporalExecutor(ctx, a.wrappedAgent)
	return a.wrappedAgent.ExecuteWithExecutor(context.Background(), in, executor)
}

func (a *TemporalAgent) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	return a.wrappedAgent.LoadMessages(ctx, namespace, previousMessageID)
}

func (a *TemporalAgent) SaveMessages(ctx context.Context, namespace string, msgId string, previousMsgId string, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return a.wrappedAgent.SaveMessages(ctx, namespace, msgId, previousMsgId, conversationId, messages, meta)
}

func (a *TemporalAgent) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return a.wrappedAgent.SaveSummary(ctx, namespace, summary)
}

func (a *TemporalAgent) GetPrompt(ctx context.Context, runContext map[string]any) (string, error) {
	return a.wrappedAgent.GetPrompt(ctx, runContext)
}

func (a *TemporalAgent) NewStreamingResponses(ctx context.Context, req *responses.Request) (*responses.Response, error) {
	info := activity.GetInfo(ctx)
	return a.wrappedAgent.NewStreamingResponses(ctx, req, func(chunk *responses.ResponseChunk) {
		a.wrappedAgent.GetStreamBroker().Publish(ctx, info.WorkflowExecution.ID, chunk)
	})
}

func (a *TemporalAgent) CallTool(ctx context.Context, toolCall *responses.FunctionCallMessage, runContext map[string]any) (*responses.FunctionCallOutputMessage, error) {
	info := activity.GetInfo(ctx)

	return a.wrappedAgent.CallTool(ctx, toolCall, runContext, func(chunk *responses.ResponseChunk) {
		a.wrappedAgent.GetStreamBroker().Publish(ctx, info.WorkflowExecution.ID, chunk)
	})
}

func (a *TemporalAgent) RunCreated(ctx context.Context, runId string, traceId string, cb func(chunk *responses.ResponseChunk)) error {
	info := activity.GetInfo(ctx)

	return a.wrappedAgent.RunCreated(ctx, runId, traceId, func(chunk *responses.ResponseChunk) {
		a.wrappedAgent.GetStreamBroker().Publish(ctx, info.WorkflowExecution.ID, chunk)
	})
}

func (a *TemporalAgent) RunPaused(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	info := activity.GetInfo(ctx)

	return a.wrappedAgent.RunPaused(ctx, runId, traceId, runState, func(chunk *responses.ResponseChunk) {
		a.wrappedAgent.GetStreamBroker().Publish(ctx, info.WorkflowExecution.ID, chunk)
	})
}

func (a *TemporalAgent) RunCompleted(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	info := activity.GetInfo(ctx)

	return a.wrappedAgent.RunCompleted(ctx, runId, traceId, runState, func(chunk *responses.ResponseChunk) {
		a.wrappedAgent.GetStreamBroker().Publish(ctx, info.WorkflowExecution.ID, chunk)
	})
}
