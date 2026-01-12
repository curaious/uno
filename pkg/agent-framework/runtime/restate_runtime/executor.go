package restate_runtime

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
	"go.opentelemetry.io/otel/attribute"
)

func NewRestateExecutor(workflowCtx restate.WorkflowContext, wrappedAgent *agents.Agent) *RestateExecutor {
	return &RestateExecutor{
		workflowCtx:  workflowCtx,
		wrappedAgent: wrappedAgent,
	}
}

type RestateExecutor struct {
	workflowCtx  restate.Context
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

func (a *RestateExecutor) NewStreamingResponses(ctx context.Context, req *responses.Request) (*responses.Response, error) {
	return restate.Run(a.workflowCtx, func(ctx restate.RunContext) (*responses.Response, error) {
		return a.wrappedAgent.NewStreamingResponses(ctx, req)
	})
}

func (a *RestateExecutor) CallTool(ctx context.Context, toolCall *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return restate.Run(a.workflowCtx, func(ctx restate.RunContext) (*responses.FunctionCallOutputMessage, error) {
		return a.wrappedAgent.CallTool(ctx, toolCall)
	})
}

// StartSpan is a no-op for Restate workflows.
// Restate handles workflow tracing via its framework-level OpenTelemetry integration.
// Spans inside restate.Run() blocks (Agent.NewStreamingResponses, Agent.CallTool) provide detailed tracing.
func (a *RestateExecutor) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	return ctx, func() {}
}
