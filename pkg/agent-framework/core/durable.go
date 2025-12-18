package core

import "context"

// DurableExecutor provides an abstraction for durable/checkpointed execution.
// Implementations can provide different durability guarantees:
// - NoOpExecutor: No durability (default, existing behavior)
// - RestateExecutor: Durable via Restate
// - TemporalExecutor: Durable via Temporal (future)
// - RedisExecutor: Durable via Redis Streams (future)
type DurableExecutor interface {
	// Run executes a function with durability guarantees.
	// If the process crashes after fn() completes, the result is restored
	// from the checkpoint instead of re-executing fn().
	//
	// The name parameter is used to identify the checkpoint (must be unique per step).
	Run(ctx context.Context, name string, fn func(ctx context.Context) (any, error)) (any, error)

	// RunT is a type-safe version of Run using generics.
	// Use this when you need type safety.
	// RunT[T any](ctx context.Context, name string, fn func(ctx context.Context) (T, error)) (T, error)

	// Set stores a value in durable state.
	// The value persists across crashes and can be retrieved with Get().
	Set(ctx context.Context, key string, value any) error

	// Get retrieves a value from durable state.
	// Returns the value and true if found, or nil and false if not found.
	Get(ctx context.Context, key string) (any, bool, error)

	// Checkpoint creates an explicit checkpoint marker.
	// Useful for marking progress without storing data.
	Checkpoint(ctx context.Context, name string) error
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

// Set is a no-op for the NoOpExecutor.
func (e *NoOpExecutor) Set(ctx context.Context, key string, value any) error {
	return nil // No-op
}

// Get always returns not found for the NoOpExecutor.
func (e *NoOpExecutor) Get(ctx context.Context, key string) (any, bool, error) {
	return nil, false, nil // No state persistence
}

// Checkpoint is a no-op for the NoOpExecutor.
func (e *NoOpExecutor) Checkpoint(ctx context.Context, name string) error {
	return nil // No-op
}

// Ensure NoOpExecutor implements DurableExecutor
var _ DurableExecutor = (*NoOpExecutor)(nil)

// DurableRun is a helper function for type-safe durable execution.
// Usage: result, err := DurableRun[MyType](ctx, executor, "step-name", func(ctx) (MyType, error) { ... })
func DurableRun[T any](ctx context.Context, executor DurableExecutor, name string, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	result, err := executor.Run(ctx, name, func(ctx context.Context) (any, error) {
		return fn(ctx)
	})
	if err != nil {
		return zero, err
	}

	// Type assertion
	if result == nil {
		return zero, nil
	}

	typed, ok := result.(T)
	if !ok {
		return zero, nil
	}

	return typed, nil
}
