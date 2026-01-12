package restate_runtime

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/agents"
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

// Ensure RestateExecutor implements DurableExecutor
var _ agents.DurableExecutor = (*RestateExecutor)(nil)
