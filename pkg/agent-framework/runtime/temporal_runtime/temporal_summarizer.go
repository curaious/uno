package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/workflow"
)

type TemporalConversationSummarizer struct {
	wrappedSummarizer core.HistorySummarizer
}

func NewTemporalConversationSummarizer(wrappedSummarizer core.HistorySummarizer) *TemporalConversationSummarizer {
	return &TemporalConversationSummarizer{wrappedSummarizer: wrappedSummarizer}
}

func (t *TemporalConversationSummarizer) Summarize(ctx context.Context, msgIdToRunId map[string]string, messages []responses.InputMessageUnion, usage *responses.Usage) (*core.SummaryResult, error) {
	return t.wrappedSummarizer.Summarize(ctx, msgIdToRunId, messages, usage)
}

type TemporalConversationSummarizerProxy struct {
	workflowCtx workflow.Context
	prefix      string
}

func NewTemporalConversationSummarizerProxy(workflowCtx workflow.Context, prefix string) core.HistorySummarizer {
	return &TemporalConversationSummarizerProxy{
		workflowCtx: workflowCtx,
		prefix:      prefix,
	}
}

func (t *TemporalConversationSummarizerProxy) Summarize(ctx context.Context, msgIdToRunId map[string]string, messages []responses.InputMessageUnion, usage *responses.Usage) (*core.SummaryResult, error) {
	var summaryResult *core.SummaryResult
	err := workflow.ExecuteActivity(t.workflowCtx, t.prefix+"_SummarizerActivity", msgIdToRunId, messages, usage).Get(t.workflowCtx, &summaryResult)
	if err != nil {
		return nil, err
	}

	return summaryResult, nil
}
