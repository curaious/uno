package restate_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
	"github.com/restatedev/sdk-go/ingress"
)

// WorkflowInput is the input structure for the Restate workflow.
type WorkflowInput struct {
	AgentName string `json:"agent_name"`

	Namespace         string
	PreviousMessageID string
	Messages          []responses.InputMessageUnion
	RunContext        map[string]any
}

// RestateRuntime executes agents via Restate workflows for durability.
// It registers the agent in the global registry and invokes a Restate workflow
// that reconstructs the agent with RestateExecutor for crash recovery.
type RestateRuntime struct {
	client *ingress.Client
}

// NewRestateRuntime creates a new Restate runtime.
// The agentName is used to look up the agent config inside the workflow.
func NewRestateRuntime(endpoint string) *RestateRuntime {
	client := ingress.NewClient(endpoint)
	return &RestateRuntime{
		client: client,
	}
}

// Run registers the agent in the global registry and invokes the Restate workflow.
func (r *RestateRuntime) Run(ctx context.Context, agent *agents.Agent, in *agents.AgentInput) (*agents.AgentOutput, error) {
	// Invoke workflow with agent name and messages
	runID := uuid.NewString()
	input := &WorkflowInput{
		AgentName:         agent.Name(),
		Namespace:         in.Namespace,
		PreviousMessageID: in.PreviousMessageID,
		Messages:          in.Messages,
		RunContext:        in.RunContext,
	}

	in.Callback = nil

	return ingress.Workflow[*WorkflowInput, *agents.AgentOutput](
		r.client,
		"AgentWorkflow",
		runID,
		"Run",
	).Request(ctx, input)
}
