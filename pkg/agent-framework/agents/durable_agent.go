package agents

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
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
	name                string
	output              any
	history             core.ChatHistory
	instruction         string
	instructionProvider core.SystemPromptProvider
	tools               []core.Tool
	llm                 llm.Provider
	executor            core.DurableExecutor
	maxLoops            int
	parameters          responses.Parameters
}

// DurableAgentOptions configures the DurableAgent.
type DurableAgentOptions struct {
	// Existing agent options
	History             core.ChatHistory
	Instruction         string
	InstructionProvider core.SystemPromptProvider
	Name                string
	LLM                 llm.Provider
	Output              any
	Tools               []core.Tool
	Parameters          responses.Parameters

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
		name:                opts.Name,
		output:              opts.Output,
		history:             opts.History,
		instruction:         opts.Instruction,
		instructionProvider: opts.InstructionProvider,
		tools:               opts.Tools,
		llm:                 opts.LLM,
		executor:            executor,
		maxLoops:            maxLoops,
	}, nil
}

// Execute runs the agent with durable execution.
// Each LLM call and tool execution is checkpointed.
func (e *DurableAgent) Execute(ctx context.Context, msgs []responses.InputMessageUnion, cb func(chunk *responses.ResponseChunk)) ([]responses.OutputMessageUnion, error) {
	ctx, span := tracer.Start(ctx, "DurableAgent.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("agent.name", e.name))

	// If history is not enabled, set a default history without persistence.
	// This is required for the agent loop to work.
	if e.history == nil {
		e.history = history.NewConversationManager(nil, "none", "")
	}

	finalOutput := []responses.OutputMessageUnion{}
	var err error
	runUsage := responses.Usage{}

	// Load the conversation history
	_, err = e.history.LoadMessages(ctx)
	if err != nil {
		span.RecordError(err)
		return finalOutput, err
	}

	// Add the incoming new message to the conversation
	e.history.AddMessages(ctx, msgs, nil)

	// Set up the system instruction
	instruction := "You are a helpful assistant."
	if e.instruction != "" {
		instruction = e.instruction
	} else if e.instructionProvider != nil {
		instruction, err = e.instructionProvider.GetPrompt(ctx, msgs)
		if err != nil {
			return finalOutput, err
		}
	}

	tools := []responses.ToolUnion{}
	for _, tool := range e.tools {
		if t := tool.Tool(ctx); t != nil {
			tools = append(tools, *t)
		}
	}

	cb(&responses.ResponseChunk{
		OfRunCreated: &responses.ChunkResponse[constants.ChunkTypeRunCreated]{
			Response: responses.ChunkResponseData{
				Object: "run",
			},
		},
	})

	loopCount := 0

	// Agent executor loop
	for loopCount < e.maxLoops {
		loopCount++
		// Check for cancellation
		if cancelled, ok, _ := e.executor.Get(ctx, "cancelled"); ok && cancelled.(bool) {
			slog.InfoContext(ctx, "agent execution cancelled")
			return finalOutput, fmt.Errorf("execution cancelled")
		}

		// Get the messages from the conversation history
		convMessages, err := e.history.GetMessages(ctx)
		if err != nil {
			span.RecordError(err)
			return finalOutput, err
		}

		// DURABLE CHECKPOINT: LLM Call
		respAny, err := e.executor.Run(ctx, fmt.Sprintf("llm-call-%d", loopCount), func(ctx context.Context) (any, error) {
			stream, err := e.llm.NewStreamingResponses(ctx, &responses.Request{
				Instructions: utils.Ptr(instruction),
				Input: responses.InputUnion{
					OfInputMessageList: convMessages,
				},
				Tools: tools,
				//OutputFormat: e.output,
				Parameters: e.parameters,
			})
			if err != nil {
				return finalOutput, err
			}

			acc := Accumulator{}
			return acc.ReadStream(stream, cb)
		})
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		buf, err := sonic.Marshal(respAny)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		resp := &responses.Response{}
		if err := sonic.Unmarshal(buf, resp); err != nil {
			span.RecordError(err)
			return nil, err
		}

		finalOutput = resp.Output

		// Track the LLM's usage
		runUsage.InputTokens += resp.Usage.InputTokens
		runUsage.OutputTokens += resp.Usage.OutputTokens
		runUsage.InputTokensDetails.CachedTokens += resp.Usage.InputTokensDetails.CachedTokens
		runUsage.TotalTokens += resp.Usage.TotalTokens

		// Execute tool calls
		toolResults := []*responses.FunctionCallOutputMessage{}
		hasToolCalls := false

		for _, msg := range finalOutput {
			if msg.OfFunctionCall == nil {
				continue
			}

			hasToolCalls = true

			args := map[string]interface{}{}
			if err := sonic.Unmarshal([]byte(msg.OfFunctionCall.Arguments), &args); err != nil {
				return finalOutput, err
			}

			for _, tool := range e.tools {
				toolResultAny, err := e.executor.Run(ctx, fmt.Sprintf("tool-%s-%s", msg.OfFunctionCall.ID, msg.OfFunctionCall.Name), func(ctx context.Context) (any, error) {
					if msg.OfFunctionCall.Name == tool.Tool(ctx).OfFunction.Name {
						toolResult, err := tool.Execute(ctx, msg.OfFunctionCall)
						if err != nil {
							span.RecordError(err)
							slog.ErrorContext(ctx, "tool call execution failed", slog.Any("error", err))
							return nil, err
						}

						return toolResult, nil
					}

					return nil, fmt.Errorf("tool %s not found", msg.OfFunctionCall.Name)
				})
				if err != nil {
					span.RecordError(err)
					slog.ErrorContext(ctx, "tool call execution failed", slog.Any("error", err))
				}

				buf, err = sonic.Marshal(toolResultAny)
				if err != nil {
					span.RecordError(err)
					return nil, err
				}

				toolResult := &responses.FunctionCallOutputMessage{}
				if err := sonic.Unmarshal(buf, toolResult); err != nil {
					span.RecordError(err)
					return nil, err
				}

				toolResults = append(toolResults, toolResult)
				cb(&responses.ResponseChunk{
					OfFunctionCallOutput: toolResult,
				})
			}
		}

		inputMsgs := []responses.InputMessageUnion{}
		for _, outMsg := range resp.Output {
			inputMsg, err := outMsg.AsInput()
			if err != nil {
				slog.ErrorContext(ctx, "output msg conversion failed", slog.Any("error", err))
				return finalOutput, err
			}
			inputMsgs = append(inputMsgs, inputMsg)
		}

		// Put final output into the conversation
		e.history.AddMessages(ctx, inputMsgs, resp.Usage)

		// Exit if no tool calls
		if !hasToolCalls {
			break
		}

		// Put tool results into the conversation
		nativeToolResults := []responses.InputMessageUnion{}
		for _, toolResult := range toolResults {
			nativeToolResults = append(nativeToolResults, responses.InputMessageUnion{
				OfFunctionCallOutput: &responses.FunctionCallOutputMessage{
					ID:     toolResult.ID,
					CallID: toolResult.CallID,
					Output: responses.FunctionCallOutputContentUnion{
						OfString: toolResult.Output.OfString,
					},
				},
			})
		}

		// Put tool results into the conversation
		e.history.AddMessages(ctx, nativeToolResults, resp.Usage)
	}

	// Run end
	cb(&responses.ResponseChunk{
		OfRunCreated: &responses.ChunkResponse[constants.ChunkTypeRunCreated]{
			Response: responses.ChunkResponseData{
				Object: "run",
				Usage:  runUsage,
			},
		},
	})

	if loopCount >= e.maxLoops {
		return finalOutput, fmt.Errorf("exceeded maximum loops (%d)", e.maxLoops)
	}

	// Save the conversation history
	err = e.history.SaveMessages(ctx, map[string]any{
		"usage": runUsage,
	})
	if err != nil {
		span.RecordError(err)
		return finalOutput, err
	}

	return finalOutput, nil
}

// Cancel signals the agent to stop execution.
func (e *DurableAgent) Cancel(ctx context.Context, reason string) error {
	return nil
}
