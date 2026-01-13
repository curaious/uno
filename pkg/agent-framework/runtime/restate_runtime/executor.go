package restate_runtime

import (
	"context"
	"fmt"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
	"go.opentelemetry.io/otel/attribute"
)

// NewRestateExecutor creates a new RestateExecutor with optional stream broker.
func NewRestateExecutor(workflowCtx restate.WorkflowContext, wrappedAgent *agents.Agent) *RestateExecutor {
	return &RestateExecutor{
		workflowCtx:  workflowCtx,
		wrappedAgent: wrappedAgent,
	}
}

type RestateExecutor struct {
	workflowCtx  restate.WorkflowContext
	wrappedAgent *agents.Agent
}

func (a *RestateExecutor) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	return restate.Run[[]conversation.ConversationMessage](a.workflowCtx, func(ctx restate.RunContext) ([]conversation.ConversationMessage, error) {
		return a.wrappedAgent.LoadMessages(ctx, namespace, previousMessageID)
	})
}

func (a *RestateExecutor) SaveMessages(ctx context.Context, namespace string, msgId string, previousMsgId string, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	_, err := restate.Run(a.workflowCtx, func(ctx restate.RunContext) (string, error) {
		return "", a.wrappedAgent.SaveMessages(ctx, namespace, msgId, previousMsgId, conversationId, messages, meta)
	})

	return err
}

func (a *RestateExecutor) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	_, err := restate.Run(a.workflowCtx, func(ctx restate.RunContext) (string, error) {
		return "", a.wrappedAgent.SaveSummary(ctx, namespace, summary)
	})

	return err
}

func (a *RestateExecutor) GetPrompt(ctx context.Context, runContext map[string]any) (string, error) {
	return restate.Run(a.workflowCtx, func(ctx restate.RunContext) (string, error) {
		return a.wrappedAgent.GetPrompt(ctx, runContext)
	})
}

func (a *RestateExecutor) NewStreamingResponses(ctx context.Context, req *responses.Request, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error) {
	return restate.Run(a.workflowCtx, func(runCtx restate.RunContext) (*responses.Response, error) {
		// Re-inject the streaming callback into the new context
		return a.wrappedAgent.NewStreamingResponses(runCtx, req, func(chunk *responses.ResponseChunk) {
			a.wrappedAgent.GetStreamBroker().Publish(ctx, a.GetRunID(ctx), chunk)
		})
	})
}

func (a *RestateExecutor) CallTool(ctx context.Context, toolCall *responses.FunctionCallMessage, runContext map[string]any, cb func(chunk *responses.ResponseChunk)) (*responses.FunctionCallOutputMessage, error) {
	return restate.Run(a.workflowCtx, func(ctx restate.RunContext) (*responses.FunctionCallOutputMessage, error) {
		return a.wrappedAgent.CallTool(ctx, toolCall, runContext, func(chunk *responses.ResponseChunk) {
			a.wrappedAgent.GetStreamBroker().Publish(ctx, a.GetRunID(ctx), chunk)
		})
	})
}

func (a *RestateExecutor) RunCreated(ctx context.Context, runId string, traceId string, cb func(chunk *responses.ResponseChunk)) error {
	_, err := restate.Run(a.workflowCtx, func(ctx restate.RunContext) (string, error) {
		return "", a.wrappedAgent.RunCreated(ctx, runId, traceId, func(chunk *responses.ResponseChunk) {
			fmt.Println("publishing chunk from RunCreated", runId, chunk)
			a.wrappedAgent.GetStreamBroker().Publish(ctx, a.GetRunID(ctx), chunk)
		})
	})
	return err
}

func (a *RestateExecutor) RunPaused(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	_, err := restate.Run(a.workflowCtx, func(ctx restate.RunContext) (string, error) {
		return "", a.wrappedAgent.RunPaused(ctx, runId, traceId, runState, func(chunk *responses.ResponseChunk) {
			a.wrappedAgent.GetStreamBroker().Publish(ctx, a.GetRunID(ctx), chunk)
		})
	})
	return err
}

func (a *RestateExecutor) RunCompleted(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	_, err := restate.Run(a.workflowCtx, func(ctx restate.RunContext) (string, error) {
		return "", a.wrappedAgent.RunCompleted(ctx, runId, traceId, runState, func(chunk *responses.ResponseChunk) {
			a.wrappedAgent.GetStreamBroker().Publish(ctx, a.GetRunID(ctx), chunk)
		})
	})
	return err
}

func (a *RestateExecutor) GetRunID(ctx context.Context) string {
	return restate.Key(a.workflowCtx)
}

// StartSpan is a no-op for Restate workflows.
func (a *RestateExecutor) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	return ctx, func() {}
}
