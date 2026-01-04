# Example 15: Temporal Agent Runtime

This example demonstrates the **reference implementation** of Temporal integration with the Uno Agent Framework.

## ⚠️ Important Notice

**This example does NOT work out-of-the-box** and is provided as:
1. A reference implementation showing the integration pattern
2. Documentation of what needs to be implemented
3. A starting point for completing the Temporal integration

## Why It Doesn't Work

Temporal's `workflow.Context` is incompatible with Go's standard `context.Context`, which means:

- ❌ LLM API calls don't work in workflows
- ❌ Database operations don't work in workflows
- ❌ HTTP requests don't work in workflows
- ❌ Any I/O operation will fail

## What's Provided

This example shows you:

✅ How to create a Temporal client
✅ How to configure an agent with TemporalRuntime
✅ How to register workflows with workers
✅ The structure needed for a working implementation

## What's Missing

To make this work, you need to implement:

1. **Activities for all I/O operations**:
   - `LLMInferenceActivity` - LLM API calls
   - `LoadConversationActivity` - DB reads
   - `SaveConversationActivity` - DB writes
   - `ExecuteToolActivity` - Tool executions

2. **Refactor DurableAgent.Execute()** to use Activities instead of direct I/O

3. **Update TemporalExecutor** to call Activities

## Running the Example

```bash
# Start Temporal server (requires Docker)
docker compose up temporal

# In another terminal, run the example
cd examples/15_temporal_agent
go run main.go
```

You'll see output explaining the limitations and what needs to be implemented.

## For Working Durable Agents

If you need durable agent execution today, use:

- **RestateRuntime**: Fully supported, production-ready
- **LocalRuntime**: No durability, but works immediately

## Learn More

See the comprehensive documentation at:
- `pkg/agent-framework/providers/temporal/README.md`

This explains:
- The fundamental context incompatibility issue
- Detailed refactoring steps needed
- Why Restate works better with the current architecture
- How to contribute a full implementation

## Example Output

```
=================================================================
Temporal Agent Runtime - Reference Implementation
=================================================================

⚠️  This example demonstrates the integration pattern but requires
   significant refactoring to work. See the README for details:

   pkg/agent-framework/providers/temporal/README.md

=================================================================

✓ Temporal client connected
✓ Agent configured
✓ Workflow registered

❌ Activities NOT implemented - execution will fail

To complete this implementation:
  1. Create Activity functions for all I/O operations
  2. Register Activities with the worker
  3. Modify DurableAgent to use Activities
  4. Update TemporalExecutor to call Activities

For working durable agent execution, use RestateRuntime instead.
```

## Contributing

Want to complete the Temporal integration? See the refactoring guide in:
`pkg/agent-framework/providers/temporal/README.md`

This would be a valuable contribution to the project!

