package restate_runtime

import (
	"fmt"

	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	restate "github.com/restatedev/sdk-go"
)

// AgentWorkflow is the Restate workflow that executes agents with durability.
type AgentWorkflow struct{}

// Run executes the agent inside a Restate workflow context.
func (w AgentWorkflow) Run(restateCtx restate.WorkflowContext, input *WorkflowInput) (*agents.AgentOutput, error) {
	agent := agents.GetAgent(input.AgentName)
	if agent == nil {
		return &agents.AgentOutput{Status: core.RunStatusError}, fmt.Errorf("agent not found: %s", input.AgentName)
	}

	// Create RestateExecutor from workflow context with optional stream broker
	executor := NewRestateExecutor(restateCtx, agent)

	// Execute using the SAME agent instance with durability
	return agent.ExecuteWithExecutor(restateCtx, &agents.AgentInput{
		Namespace:         input.Namespace,
		PreviousMessageID: input.PreviousMessageID,
		Messages:          input.Messages,
		RunContext:        input.RunContext,
	}, executor)
}
