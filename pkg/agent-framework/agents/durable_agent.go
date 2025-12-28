package agents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/constants"
	"github.com/praveen001/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel/attribute"
)

// DurableAgent wraps an Agent with durable execution capabilities.
// It checkpoints after each LLM call and tool execution, allowing
// the agent to resume from the last checkpoint after a crash.
//
// If no executor is provided, it falls back to NoOpExecutor (no durability).
type DurableAgent struct {
	name        string
	output      any
	history     core.ChatHistory
	instruction core.SystemPromptProvider
	tools       []core.Tool
	llm         llm.Provider
	executor    core.DurableExecutor
	maxLoops    int
	parameters  responses.Parameters
}

// DurableAgentOptions configures the DurableAgent.
type DurableAgentOptions struct {
	History     core.ChatHistory
	Instruction core.SystemPromptProvider
	Name        string
	LLM         llm.Provider
	Output      any
	Tools       []core.Tool
	Parameters  responses.Parameters

	// Durability options
	Executor core.DurableExecutor // If nil, uses NoOpExecutor
	MaxLoops int                  // Maximum loop iterations (default: 50)
}

// NewDurableAgent creates a new agent with optional durable execution.
func NewDurableAgent(opts *DurableAgentOptions) (*DurableAgent, error) {
	executor := opts.Executor
	if executor == nil {
		executor = core.NewNoOpExecutor()
	}

	maxLoops := opts.MaxLoops
	if maxLoops <= 0 {
		maxLoops = 50
	}

	return &DurableAgent{
		name:        opts.Name,
		output:      opts.Output,
		history:     opts.History,
		instruction: opts.Instruction,
		tools:       opts.Tools,
		llm:         opts.LLM,
		executor:    executor,
		maxLoops:    maxLoops,
		parameters:  opts.Parameters,
	}, nil
}

// Execute runs the agent with durable execution.
// Each LLM call and tool execution is checkpointed.
// Supports human-in-the-loop via the pause/resume pattern.
func (e *DurableAgent) Execute(ctx context.Context, msgs []responses.InputMessageUnion, cb func(chunk *responses.ResponseChunk)) (*ExecutionResult, error) {
	ctx, span := tracer.Start(ctx, "DurableAgent.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("agent.name", e.name))

	// If history is not enabled, set a default history without persistence.
	// This is required for the agent loop to work.
	if e.history == nil {
		// DURABLE: Generate message ID deterministically
		msgIdAny, err := e.executor.Run(ctx, "generate-message-id", func(ctx context.Context) (any, error) {
			return uuid.NewString(), nil
		})
		if err != nil {
			span.RecordError(err)
			return &ExecutionResult{Status: core.RunStatusError}, err
		}
		msgId := msgIdAny.(string)
		e.history = history.NewConversationManager(nil, "none", "", history.WithMessageID(msgId))
	}

	// DURABLE CHECKPOINT: Load the conversation history (DB read)
	_, err := e.executor.Run(ctx, "load-messages", func(ctx context.Context) (any, error) {
		return e.history.LoadMessages(ctx)
	})
	if err != nil {
		span.RecordError(err)
		return &ExecutionResult{Status: core.RunStatusError}, err
	}

	// Load run state from meta (in-memory, no DB call)
	runId := e.history.GetMessageID()
	meta := e.history.GetMeta()
	runState := core.LoadRunStateFromMeta(meta)
	var rejectedToolCallIds []string
	var traceid string
	sc := span.SpanContext()
	if sc.IsValid() {
		traceid = sc.TraceID().String()
	}

	// Initialize state
	if runState == nil || runState.IsComplete() {
		// FRESH RUN: No state or previous run completed
		runState = core.NewRunState()
		if len(msgs) > 0 {
			e.history.AddMessages(ctx, msgs, nil)
		}

		// DURABLE: Emit run created event (side effect)
		e.executor.Run(ctx, "emit-run-created", func(ctx context.Context) (any, error) {
			cb(&responses.ResponseChunk{
				OfRunCreated: &responses.ChunkRun[constants.ChunkTypeRunCreated]{
					RunState: responses.ChunkRunData{
						Id:      runId,
						Object:  "run",
						Status:  "created",
						TraceID: traceid,
					},
				},
			})
			return nil, nil
		})
	} else if runState.IsPaused() {
		// RESUME: Transition to execute the approved tools
		// msgs is ignored on resume - we continue from existing messages with pending tools
		if runState.CurrentStep == core.StepAwaitApproval {
			// Expected an approval response message
			if len(msgs) == 0 || msgs[0].OfFunctionCallApprovalResponse == nil {
				return &ExecutionResult{Status: core.RunStatusError}, errors.New("expected approval response message")
			}
			runState.CurrentStep = core.StepExecuteTools
			rejectedToolCallIds = msgs[0].OfFunctionCallApprovalResponse.RejectedCallIds
		}
	}

	// DURABLE: Emit run in progress event (side effect)
	e.executor.Run(ctx, "emit-run-in-progress", func(ctx context.Context) (any, error) {
		cb(&responses.ResponseChunk{
			OfRunInProgress: &responses.ChunkRun[constants.ChunkTypeRunInProgress]{
				RunState: responses.ChunkRunData{
					Id:      runId,
					Object:  "run",
					Status:  "in_progress",
					TraceID: traceid,
				},
			},
		})
		return nil, nil
	})

	// DURABLE CHECKPOINT: Load system instruction (potential DB read)
	instruction := "You are a helpful assistant."
	if e.instruction != nil {
		instructionAny, err := e.executor.Run(ctx, "load-instruction", func(ctx context.Context) (any, error) {
			return e.instruction.GetPrompt(ctx)
		})
		if err != nil {
			span.RecordError(err)
			return &ExecutionResult{Status: core.RunStatusError}, err
		}
		if instructionAny != nil {
			instruction = instructionAny.(string)
		}
	}

	// Collect tool definitions
	tools := []responses.ToolUnion{}
	for _, tool := range e.tools {
		if t := tool.Tool(ctx); t != nil {
			tools = append(tools, *t)
		}
	}

	finalOutput := []responses.OutputMessageUnion{}

	// Main loop - driven by state machine
	for runState.LoopIteration < e.maxLoops {
		// Check for cancellation (durable)
		if cancelled, ok, _ := e.executor.Get(ctx, "cancelled"); ok && cancelled.(bool) {
			slog.InfoContext(ctx, "agent execution cancelled")
			return &ExecutionResult{Status: core.RunStatusError}, fmt.Errorf("execution cancelled")
		}

		switch runState.NextStep() {

		case core.StepCallLLM:
			// Get the messages from the conversation history
			convMessages, err := e.history.GetMessages(ctx)
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			// DURABLE CHECKPOINT: LLM Call
			respAny, err := e.executor.Run(ctx, fmt.Sprintf("llm-call-%d", runState.LoopIteration), func(ctx context.Context) (any, error) {
				stream, err := e.llm.NewStreamingResponses(ctx, &responses.Request{
					Instructions: utils.Ptr(instruction),
					Input: responses.InputUnion{
						OfInputMessageList: convMessages,
					},
					Tools:      tools,
					Parameters: e.parameters,
				})
				if err != nil {
					return nil, err
				}

				acc := Accumulator{}
				return acc.ReadStream(stream, cb)
			})
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			// Unmarshal response
			buf, err := sonic.Marshal(respAny)
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			resp := &responses.Response{}
			if err := sonic.Unmarshal(buf, resp); err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			finalOutput = append(finalOutput, resp.Output...)

			// Track the LLM's usage
			runState.Usage.InputTokens += resp.Usage.InputTokens
			runState.Usage.OutputTokens += resp.Usage.OutputTokens
			runState.Usage.InputTokensDetails.CachedTokens += resp.Usage.InputTokensDetails.CachedTokens
			runState.Usage.TotalTokens += resp.Usage.TotalTokens

			// Convert output to input messages and add to history
			inputMsgs := []responses.InputMessageUnion{}
			for _, outMsg := range resp.Output {
				inputMsg, err := outMsg.AsInput()
				if err != nil {
					slog.ErrorContext(ctx, "output msg conversion failed", slog.Any("error", err))
					return &ExecutionResult{Status: core.RunStatusError}, err
				}
				inputMsgs = append(inputMsgs, inputMsg)
			}
			e.history.AddMessages(ctx, inputMsgs, resp.Usage)

			// Extract tool calls
			toolCalls := []responses.FunctionCallMessage{}
			for _, msg := range resp.Output {
				if msg.OfFunctionCall != nil {
					toolCalls = append(toolCalls, *msg.OfFunctionCall)
				}
			}

			if len(toolCalls) == 0 {
				// No tools = done
				runState.TransitionToComplete()
			} else {
				// Partition tools by approval requirement
				needsApproval, immediate := e.partitionByApproval(ctx, toolCalls)

				// Execute immediate tools first (if any), then handle approval
				if len(immediate) > 0 {
					runState.TransitionToExecuteTools(immediate)
					// Store tools needing approval for after immediate execution
					if len(needsApproval) > 0 {
						runState.ToolsAwaitingApproval = needsApproval
					}
				} else if len(needsApproval) > 0 {
					// Only approval-required tools, no immediate ones
					runState.TransitionToAwaitApproval(needsApproval)
				}
			}

		case core.StepExecuteTools:
			// Execute pending tool calls
			for _, toolCall := range runState.PendingToolCalls {
				tool := e.findTool(ctx, toolCall.Name)
				if tool == nil {
					slog.ErrorContext(ctx, "tool not found", slog.String("tool_name", toolCall.Name))
					continue
				}

				var toolResult *responses.FunctionCallOutputMessage

				if slices.Contains(rejectedToolCallIds, toolCall.CallID) {
					// Tool was rejected by human
					toolResult = &responses.FunctionCallOutputMessage{
						ID:     toolCall.ID,
						CallID: toolCall.CallID,
						Output: responses.FunctionCallOutputContentUnion{
							OfString: utils.Ptr("Request to call this tool has been declined"),
						},
					}
				} else {
					// DURABLE CHECKPOINT: Tool execution
					toolResultAny, err := e.executor.Run(ctx, fmt.Sprintf("tool-%s-%s", toolCall.ID, toolCall.Name), func(ctx context.Context) (any, error) {
						return tool.Execute(ctx, &toolCall)
					})
					if err != nil {
						span.RecordError(err)
						slog.ErrorContext(ctx, "tool call execution failed", slog.Any("error", err))
						return &ExecutionResult{Status: core.RunStatusError}, err
					}

					// Unmarshal tool result
					buf, err := sonic.Marshal(toolResultAny)
					if err != nil {
						span.RecordError(err)
						return &ExecutionResult{Status: core.RunStatusError}, err
					}

					toolResult = &responses.FunctionCallOutputMessage{}
					if err := sonic.Unmarshal(buf, toolResult); err != nil {
						span.RecordError(err)
						return &ExecutionResult{Status: core.RunStatusError}, err
					}
				}

				// DURABLE: Emit tool result event (side effect)
				e.executor.Run(ctx, fmt.Sprintf("emit-tool-result-%s", toolCall.ID), func(ctx context.Context) (any, error) {
					cb(&responses.ResponseChunk{
						OfFunctionCallOutput: toolResult,
					})
					return nil, nil
				})

				// Add tool result to history
				e.history.AddMessages(ctx, []responses.InputMessageUnion{
					{OfFunctionCallOutput: toolResult},
				}, nil)
			}

			runState.ClearPendingTools()

			// Check if there are tools waiting for approval (queued during immediate execution)
			if runState.HasToolsAwaitingApproval() {
				runState.PromoteAwaitingToApproval()
			} else {
				runState.TransitionToLLM()
			}

		case core.StepAwaitApproval:
			// DURABLE CHECKPOINT: Save state and exit - will resume when approval comes (DB write)
			_, err := e.executor.Run(ctx, "save-messages-paused", func(ctx context.Context) (any, error) {
				return nil, e.history.SaveMessages(ctx, runState.ToMeta(traceid))
			})
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			// DURABLE: Emit run paused event (side effect)
			e.executor.Run(ctx, "emit-run-paused", func(ctx context.Context) (any, error) {
				cb(&responses.ResponseChunk{
					OfRunPaused: &responses.ChunkRun[constants.ChunkTypeRunPaused]{
						RunState: responses.ChunkRunData{
							Id:               runId,
							Object:           "run",
							Status:           "paused",
							PendingToolCalls: runState.PendingToolCalls,
							Usage:            runState.Usage,
							TraceID:          traceid,
						},
					},
				})
				return nil, nil
			})

			return &ExecutionResult{
				Status:           core.RunStatusPaused,
				PendingApprovals: runState.PendingToolCalls,
			}, nil

		case core.StepComplete:
			// DURABLE CHECKPOINT: Save final state (DB write)
			_, err := e.executor.Run(ctx, "save-messages-complete", func(ctx context.Context) (any, error) {
				return nil, e.history.SaveMessages(ctx, runState.ToMeta(traceid))
			})
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			// DURABLE: Emit run completed event (side effect)
			e.executor.Run(ctx, "emit-run-completed", func(ctx context.Context) (any, error) {
				cb(&responses.ResponseChunk{
					OfRunCompleted: &responses.ChunkRun[constants.ChunkTypeRunCompleted]{
						RunState: responses.ChunkRunData{
							Id:      runId,
							Object:  "run",
							Status:  "completed",
							Usage:   runState.Usage,
							TraceID: traceid,
						},
					},
				})
				return nil, nil
			})

			return &ExecutionResult{
				Status: core.RunStatusCompleted,
				Output: finalOutput,
			}, nil
		}
	}

	// Max loops exceeded
	return &ExecutionResult{Status: core.RunStatusError}, fmt.Errorf("exceeded maximum loops (%d)", e.maxLoops)
}

// partitionByApproval splits tool calls into those needing approval and those that can execute immediately
func (e *DurableAgent) partitionByApproval(ctx context.Context, toolCalls []responses.FunctionCallMessage) (needsApproval []responses.FunctionCallMessage, immediate []responses.FunctionCallMessage) {
	for _, toolCall := range toolCalls {
		tool := e.findTool(ctx, toolCall.Name)
		if tool != nil && tool.NeedApproval() {
			needsApproval = append(needsApproval, toolCall)
		} else {
			immediate = append(immediate, toolCall)
		}
	}
	return needsApproval, immediate
}

// findTool finds a tool by name
func (e *DurableAgent) findTool(ctx context.Context, toolName string) core.Tool {
	for _, tool := range e.tools {
		if t := tool.Tool(ctx); t != nil && t.OfFunction != nil && t.OfFunction.Name == toolName {
			return tool
		}
	}
	return nil
}

// Cancel signals the agent to stop execution.
func (e *DurableAgent) Cancel(ctx context.Context, reason string) error {
	return e.executor.Set(ctx, "cancelled", true)
}
