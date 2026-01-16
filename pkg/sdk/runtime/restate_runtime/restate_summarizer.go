package restate_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
)

type RestateConversationSummarizer struct {
	restateCtx        restate.WorkflowContext
	wrappedSummarizer core.HistorySummarizer
}

func NewRestateConversationSummarizer(restateCtx restate.WorkflowContext, wrappedSummarizer core.HistorySummarizer) *RestateConversationSummarizer {
	return &RestateConversationSummarizer{
		restateCtx:        restateCtx,
		wrappedSummarizer: wrappedSummarizer,
	}
}

func (t *RestateConversationSummarizer) Summarize(ctx context.Context, msgIdToRunId map[string]string, messages []responses.InputMessageUnion, usage *responses.Usage) (*core.SummaryResult, error) {
	return restate.Run(t.restateCtx, func(ctx restate.RunContext) (*core.SummaryResult, error) {
		return t.wrappedSummarizer.Summarize(ctx, msgIdToRunId, messages, usage)
	})
}
