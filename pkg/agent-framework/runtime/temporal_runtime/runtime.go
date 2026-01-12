package temporal_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/agents"
	"go.temporal.io/sdk/client"
)

type TemporalRuntime struct {
	client client.Client
}

func NewTemporalRuntime() *TemporalRuntime {
	c, err := client.Dial(client.Options{})
	if err != nil {
		panic("unable to create temporal client")
	}

	return &TemporalRuntime{
		client: c,
	}
}

func (r *TemporalRuntime) Run(ctx context.Context, agent *agents.Agent, in *agents.AgentInput) (*agents.AgentOutput, error) {
	run, err := r.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: "AgentWorkflowTaskQueue",
	}, agent.Name()+"_AgentWorkflow", agent.Name(), in)
	if err != nil {
		return nil, err
	}

	// Wait for result
	var result agents.AgentOutput
	if err := run.Get(ctx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
