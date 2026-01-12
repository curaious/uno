package agents

import (
	"context"
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

// NewLocalRuntime returns the default runtime (LocalRuntime).
// This is used when no runtime is explicitly set on an agent.
func NewLocalRuntime() AgentRuntime {
	return &LocalRuntime{}
}

// Run executes the agent using ExecuteWithExecutor with NoOpExecutor.
// This provides the agent loop without durability.
// The agent instance is reused across multiple runs (cached MCP connections, resolved tools).
func (r *LocalRuntime) Run(ctx context.Context, agent *Agent, in *AgentInput) (*AgentOutput, error) {
	// Execute directly on the agent with no durability
	return agent.ExecuteWithExecutor(ctx, in, NewNoOpExecutor())
}

// DurableExecutor provides an abstraction for durable/checkpointed execution.
// Implementations can provide different durability guarantees:
// - NoOpExecutor: No durability (default, existing behavior)
// - RestateExecutor: Durable via Restate
// - TemporalExecutor: Durable via Temporal (future)
type DurableExecutor interface {
	Run(ctx context.Context, name string, fn func(ctx context.Context) (any, error)) (any, error)
}

// NoOpExecutor is the default executor with no durability.
// It simply executes functions directly without checkpointing.
// Use this for existing behavior or when durability is not needed.
type NoOpExecutor struct{}

// NewNoOpExecutor creates a new no-op executor (existing behavior).
func NewNoOpExecutor() *NoOpExecutor {
	return &NoOpExecutor{}
}

// Run executes the function directly without checkpointing.
func (e *NoOpExecutor) Run(ctx context.Context, name string, fn func(ctx context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

// Ensure NoOpExecutor implements DurableExecutor
var _ DurableExecutor = (*NoOpExecutor)(nil)
