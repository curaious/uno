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
// It looks up the agent from the registry and uses ExecuteWithExecutor
// with RestateExecutor for crash recovery.
// The agent is cached in the registry, so MCP connections are reused.
func (w AgentWorkflow) Run(restateCtx restate.WorkflowContext, input *WorkflowInput) (*agents.AgentOutput, error) {
	// Lookup agent from registry (cached, with prepared MCP connections)
	agent := agents.GetAgent(input.AgentName)
	if agent == nil {
		return &agents.AgentOutput{Status: core.RunStatusError}, fmt.Errorf("agent not found: %s", input.AgentName)
	}

	// Create RestateExecutor from workflow context
	executor := NewRestateExecutor(restateCtx, agent)

	// Execute using the SAME agent instance with durability
	// Note: The callback won't work across process boundaries, so we use a no-op callback
	// For streaming, we'd need Redis pub/sub or similar mechanism
	return agent.ExecuteWithExecutor(restateCtx, &agents.AgentInput{
		Namespace:         input.Namespace,
		PreviousMessageID: input.PreviousMessageID,
		Messages:          input.Messages,
		RunContext:        input.RunContext,
	}, executor)
}
