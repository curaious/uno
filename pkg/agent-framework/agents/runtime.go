package agents

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	restateExec "github.com/praveen001/uno/pkg/agent-framework/providers/restate"
	"github.com/praveen001/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
)

// AgentRuntime defines the interface for agent execution strategies.
// All runtimes receive the agent configuration and execute it using
// their specific strategy (local, Restate, Temporal, etc.).
type AgentRuntime interface {
	Run(ctx context.Context, agent *Agent, in *AgentInput) (*AgentOutput, error)
}

// LocalRuntime executes agents in-process with no durability.
// It uses the agent's ExecuteWithExecutor method with NoOpExecutor,
// providing the same agent loop logic but without crash recovery.
type LocalRuntime struct{}

// DefaultRuntime returns the default runtime (LocalRuntime).
// This is used when no runtime is explicitly set on an agent.
func DefaultRuntime() AgentRuntime {
	return &LocalRuntime{}
}

// Run executes the agent using ExecuteWithExecutor with NoOpExecutor.
// This provides the agent loop without durability.
// The agent instance is reused across multiple runs (cached MCP connections, resolved tools).
func (r *LocalRuntime) Run(ctx context.Context, agent *Agent, in *AgentInput) (*AgentOutput, error) {
	// Execute directly on the agent with no durability
	return agent.ExecuteWithExecutor(ctx, in, core.NewNoOpExecutor())
}

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
func (r *RestateRuntime) Run(ctx context.Context, agent *Agent, in *AgentInput) (*AgentOutput, error) {
	// Invoke workflow with agent name and messages
	runID := uuid.NewString()
	input := &WorkflowInput{
		AgentName:         agent.name,
		Namespace:         in.Namespace,
		PreviousMessageID: in.PreviousMessageID,
		Messages:          in.Messages,
		RunContext:        in.RunContext,
	}

	in.Callback = nil

	return ingress.Workflow[*WorkflowInput, *AgentOutput](
		r.client,
		"AgentWorkflow",
		runID,
		"Run",
	).Request(ctx, input)
}

// AgentWorkflow is the Restate workflow that executes agents with durability.
type AgentWorkflow struct{}

// Run executes the agent inside a Restate workflow context.
// It looks up the agent from the registry and uses ExecuteWithExecutor
// with RestateExecutor for crash recovery.
// The agent is cached in the registry, so MCP connections are reused.
func (w AgentWorkflow) Run(restateCtx restate.WorkflowContext, input *WorkflowInput) (*AgentOutput, error) {
	// Lookup agent from registry (cached, with prepared MCP connections)
	agent := GetAgent(input.AgentName)
	if agent == nil {
		return &AgentOutput{Status: core.RunStatusError}, fmt.Errorf("agent not found: %s", input.AgentName)
	}

	// Create RestateExecutor from workflow context
	executor := restateExec.NewRestateExecutor(restateCtx)

	// Execute using the SAME agent instance with durability
	// Note: The callback won't work across process boundaries, so we use a no-op callback
	// For streaming, we'd need Redis pub/sub or similar mechanism
	return agent.ExecuteWithExecutor(restateCtx, &AgentInput{
		Namespace:         input.Namespace,
		PreviousMessageID: input.PreviousMessageID,
		Messages:          input.Messages,
		RunContext:        input.RunContext,
	}, executor)
}
