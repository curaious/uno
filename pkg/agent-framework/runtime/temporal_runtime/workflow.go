package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/agents"
	"go.temporal.io/sdk/workflow"
)

type TemporalAgent struct {
	wrappedAgent *agents.Agent
}

func NewTemporalAgent(wrappedAgent *agents.Agent) *TemporalAgent {
	return &TemporalAgent{
		wrappedAgent: wrappedAgent,
	}
}

func (a *TemporalAgent) Execute(ctx workflow.Context, in *agents.AgentInput) (*agents.AgentOutput, error) {
	executor := &TemporalExecutor{workflowCtx: ctx, name: a.wrappedAgent.Name()}
	return a.wrappedAgent.ExecuteWithExecutor(context.Background(), in, executor)
}
