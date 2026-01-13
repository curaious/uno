package temporal_runtime

import (
	"context"
	"time"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel/attribute"
	"go.temporal.io/sdk/workflow"
)

type TemporalExecutor struct {
	workflowCtx  workflow.Context
	wrappedAgent *agents.Agent
}

// NewTemporalExecutor creates a new TemporalExecutor with optional stream broker.
func NewTemporalExecutor(workflowCtx workflow.Context, wrappedAgent *agents.Agent) *TemporalExecutor {
	return &TemporalExecutor{
		workflowCtx: workflow.WithActivityOptions(workflowCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		}),
		wrappedAgent: wrappedAgent,
	}
}

func (a *TemporalExecutor) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	var messages []conversation.ConversationMessage
	err := workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_LoadMessagesActivity", namespace, previousMessageID).Get(a.workflowCtx, &messages)
	if err != nil {
		return messages, err
	}

	return messages, nil
}

func (a *TemporalExecutor) SaveMessages(ctx context.Context, namespace string, msgId string, previousMsgId string, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_SaveMessagesActivity", namespace, msgId, previousMsgId, conversationId, messages, meta).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_SaveSummaryActivity", namespace, summary).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) GetPrompt(ctx context.Context, runContext map[string]any) (string, error) {
	var prompt string
	err := workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_GetPromptActivity", runContext).Get(a.workflowCtx, &prompt)
	if err != nil {
		return "", err
	}

	return prompt, nil
}

func (a *TemporalExecutor) NewStreamingResponses(ctx context.Context, req *responses.Request, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error) {
	var response *responses.Response
	err := workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_NewStreamingResponsesActivity", req).Get(a.workflowCtx, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (a *TemporalExecutor) CallTool(ctx context.Context, toolCall *responses.FunctionCallMessage, runContext map[string]any, cb func(chunk *responses.ResponseChunk)) (*responses.FunctionCallOutputMessage, error) {
	var output *responses.FunctionCallOutputMessage
	err := workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_CallToolActivity", toolCall, runContext).Get(a.workflowCtx, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (a *TemporalExecutor) RunCreated(ctx context.Context, runId string, traceId string, cb func(chunk *responses.ResponseChunk)) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_RunCreatedActivity", runId, traceId).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) RunPaused(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_RunPausedActivity", runId, traceId, runState).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) RunCompleted(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.wrappedAgent.Name()+"_RunCompletedActivity", runId, traceId, runState).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) GetRunID(ctx context.Context) string {
	return workflow.GetInfo(a.workflowCtx).WorkflowExecution.ID
}

// StartSpan is a no-op for Temporal workflows.
func (a *TemporalExecutor) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	return ctx, func() {}
}
