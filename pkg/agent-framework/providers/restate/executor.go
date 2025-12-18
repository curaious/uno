package restate

import (
	"context"
	"fmt"

	"github.com/praveen001/uno/pkg/agent-framework/core"
	restate "github.com/restatedev/sdk-go"
)

// RestateExecutor implements core.DurableExecutor using Restate.
// It wraps a Restate WorkflowContext to provide durable execution.
type RestateExecutor struct {
	ctx restate.WorkflowContext
}

// NewRestateExecutor creates a new Restate-backed durable executor.
// Pass the WorkflowContext from your Restate handler.
func NewRestateExecutor(ctx restate.WorkflowContext) *RestateExecutor {
	return &RestateExecutor{ctx: ctx}
}

// Run executes a function with Restate durability.
// If the process crashes after fn() completes, Restate will restore
// the result from its checkpoint instead of re-executing fn().
func (e *RestateExecutor) Run(ctx context.Context, name string, fn func(ctx context.Context) (any, error)) (any, error) {
	return restate.Run(e.ctx, func(runCtx restate.RunContext) (any, error) {
		// Convert restate.RunContext to context.Context for the function
		return fn(runCtx)
	})
}

// Set stores a value in Restate's durable state.
func (e *RestateExecutor) Set(ctx context.Context, key string, value any) error {
	restate.Set(e.ctx, key, value)
	return nil
}

// Get retrieves a value from Restate's durable state.
func (e *RestateExecutor) Get(ctx context.Context, key string) (any, bool, error) {
	value, err := restate.Get[any](e.ctx, key)
	if err != nil {
		return nil, false, err
	}
	return value, value != nil, nil
}

// Checkpoint creates an explicit checkpoint in Restate.
func (e *RestateExecutor) Checkpoint(ctx context.Context, name string) error {
	// In Restate, we can use a no-op Run to create a checkpoint
	_, err := restate.Run(e.ctx, func(runCtx restate.RunContext) (bool, error) {
		return true, nil // Checkpoint marker
	})
	return err
}

// Ensure RestateExecutor implements DurableExecutor
var _ core.DurableExecutor = (*RestateExecutor)(nil)

// RestateObjectExecutor implements core.DurableExecutor for Restate ObjectContext.
// Use this for Virtual Objects (stateful, keyed services).
type RestateObjectExecutor struct {
	ctx restate.ObjectContext
}

// NewRestateObjectExecutor creates a new Restate-backed durable executor for objects.
func NewRestateObjectExecutor(ctx restate.ObjectContext) *RestateObjectExecutor {
	return &RestateObjectExecutor{ctx: ctx}
}

// Run executes a function with Restate durability.
func (e *RestateObjectExecutor) Run(ctx context.Context, name string, fn func(ctx context.Context) (any, error)) (any, error) {
	return restate.Run(e.ctx, func(runCtx restate.RunContext) (any, error) {
		return fn(ctx)
	})
}

// Set stores a value in Restate's durable state.
func (e *RestateObjectExecutor) Set(ctx context.Context, key string, value any) error {
	restate.Set(e.ctx, key, value)
	return nil
}

// Get retrieves a value from Restate's durable state.
func (e *RestateObjectExecutor) Get(ctx context.Context, key string) (any, bool, error) {
	value, err := restate.Get[any](e.ctx, key)
	if err != nil {
		return nil, false, err
	}
	return value, value != nil, nil
}

// Checkpoint creates an explicit checkpoint.
func (e *RestateObjectExecutor) Checkpoint(ctx context.Context, name string) error {
	_, err := restate.Run(e.ctx, func(runCtx restate.RunContext) (bool, error) {
		return true, nil
	})
	return err
}

var _ core.DurableExecutor = (*RestateObjectExecutor)(nil)

// Helper types for typed Restate operations

// TypedRun executes a typed function with Restate durability.
// This avoids the any type assertion at the call site.
func TypedRun[T any](ctx restate.Context, fn func(restate.RunContext) (T, error)) (T, error) {
	return restate.Run(ctx, fn)
}

// WorkflowRun executes a typed function in a workflow context.
func WorkflowRun[T any](ctx restate.WorkflowContext, fn func(restate.RunContext) (T, error)) (T, error) {
	return restate.Run(ctx, fn)
}

// Helper to wrap existing agent execution in Restate

// WrapAgentExecution wraps an agent execution function with Restate durability.
// This is useful for integrating existing agent code with Restate.
//
// Example:
//
//	func (w MyWorkflow) Run(ctx restate.WorkflowContext, input AgentInput) (AgentOutput, error) {
//	    executor := restate.NewRestateExecutor(ctx)
//	    return WrapAgentExecution(ctx, executor, input, myAgent.Execute)
//	}
func WrapAgentExecution[I any, O any](
	ctx context.Context,
	executor core.DurableExecutor,
	input I,
	executeFn func(ctx context.Context, input I, executor core.DurableExecutor) (O, error),
) (O, error) {
	var zero O

	result, err := executor.Run(ctx, "agent-execution", func(ctx context.Context) (any, error) {
		return executeFn(ctx, input, executor)
	})
	if err != nil {
		return zero, err
	}

	typed, ok := result.(O)
	if !ok {
		return zero, fmt.Errorf("unexpected result type")
	}

	return typed, nil
}
