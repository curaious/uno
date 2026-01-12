package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/workflow"
)

type TemporalExecutor struct {
	workflowCtx workflow.Context
	name        string
}

func (a *TemporalExecutor) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	var messages []conversation.ConversationMessage
	err := workflow.ExecuteActivity(a.workflowCtx, a.name+"_LoadMessagesActivity", namespace, previousMessageID).Get(a.workflowCtx, &messages)
	if err != nil {
		return messages, err
	}

	return messages, nil
}

func (a *TemporalExecutor) SaveMessages(ctx context.Context, namespace string, msgId string, previousMsgId string, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.name+"_SaveMessagesActivity", namespace, msgId, previousMsgId, conversationId, messages, meta).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return workflow.ExecuteActivity(a.workflowCtx, a.name+"_SaveSummaryActivity", namespace, summary).Get(a.workflowCtx, nil)
}

func (a *TemporalExecutor) GetPrompt(ctx context.Context, runContext map[string]any) (string, error) {
	var prompt string
	err := workflow.ExecuteActivity(a.workflowCtx, a.name+"_GetPromptActivity", runContext).Get(a.workflowCtx, &prompt)
	if err != nil {
		return "", err
	}

	return prompt, nil
}

func (a *TemporalExecutor) NewStreamingResponses(ctx context.Context, req *responses.Request) (*responses.Response, error) {
	var response *responses.Response
	err := workflow.ExecuteActivity(a.workflowCtx, a.name+"_NewStreamingResponsesActivity", req).Get(a.workflowCtx, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (a *TemporalExecutor) CallTool(ctx context.Context, toolCall *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	var output *responses.FunctionCallOutputMessage
	err := workflow.ExecuteActivity(a.workflowCtx, a.name+"_CallToolActivity", toolCall).Get(a.workflowCtx, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}
