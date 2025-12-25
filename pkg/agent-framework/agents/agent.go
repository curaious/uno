package agents

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/constants"
	"github.com/praveen001/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	tracer = otel.Tracer("Agent")
)

type Agent struct {
	name        string
	output      map[string]any
	history     core.ChatHistory
	instruction core.SystemPromptProvider
	tools       []core.Tool
	llm         llm.Provider
	parameters  responses.Parameters
}

type AgentOptions struct {
	History     core.ChatHistory
	Instruction core.SystemPromptProvider
	Parameters  responses.Parameters

	Name   string
	LLM    llm.Provider
	Output map[string]any
	Tools  []core.Tool
}

func NewAgent(opts *AgentOptions) *Agent {
	if opts.Output != nil {
		format := map[string]any{
			"type":   "json_schema",
			"name":   "structured_output",
			"strict": false,
			"schema": opts.Output,
		}
		opts.Parameters.Text = &responses.TextFormat{
			Format: format,
		}
	}

	return &Agent{
		name:        opts.Name,
		output:      opts.Output,
		history:     opts.History,
		instruction: opts.Instruction,
		tools:       opts.Tools,
		llm:         opts.LLM,
		parameters:  opts.Parameters,
	}
}

// ExecutionResult represents the result of agent execution
type ExecutionResult struct {
	Status           core.RunStatus
	Output           []responses.OutputMessageUnion
	PendingApprovals []responses.FunctionCallMessage
}

func (e *Agent) Execute(ctx context.Context, msgs []responses.InputMessageUnion, cb func(*responses.ResponseChunk)) (*ExecutionResult, error) {
	ctx, span := tracer.Start(ctx, "Agent.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("agent.name", e.name))

	// If history is not enabled, set a default history without persistence.
	// This is required for the agent loop to work.
	if e.history == nil {
		e.history = history.NewConversationManager(nil, "none", "")
	}

	// Load the conversation history
	_, err := e.history.LoadMessages(ctx)
	if err != nil {
		span.RecordError(err)
		return &ExecutionResult{Status: core.RunStatusError}, err
	}

	// Load run state from meta
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

	} else if runState.IsPaused() {
		// RESUME: Transition to execute the approved tools
		// msgs is ignored on resume - we continue from existing messages with pending tools
		if runState.CurrentStep == core.StepAwaitApproval {
			// Then expected an approval response message
			if msgs[0].OfFunctionCallApprovalResponse == nil {
				return &ExecutionResult{Status: core.RunStatusError}, nil
			}
			runState.CurrentStep = core.StepExecuteTools
			rejectedToolCallIds = msgs[0].OfFunctionCallApprovalResponse.RejectedCallIds
		}
	}

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

	// Set up the system instruction
	instruction := "You are a helpful assistant."
	if e.instruction != nil {
		instruction, err = e.instruction.GetPrompt(ctx)
		if err != nil {
			return &ExecutionResult{Status: core.RunStatusError}, err
		}
	}

	tools := []responses.ToolUnion{}
	for _, tool := range e.tools {
		if t := tool.Tool(ctx); t != nil {
			tools = append(tools, *t)
		}
	}

	finalOutput := []responses.OutputMessageUnion{}
	maxLoops := 50

	// Main loop - driven by state machine
	for runState.LoopIteration < maxLoops {
		switch runState.NextStep() {

		case core.StepCallLLM:
			// Get the messages from the conversation history
			convMessages, err := e.history.GetMessages(ctx)
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			// Invoke LLM
			stream, err := e.llm.NewStreamingResponses(ctx, &responses.Request{
				Instructions: utils.Ptr(instruction),
				Input: responses.InputUnion{
					OfInputMessageList: convMessages,
				},
				Tools:      tools,
				Parameters: e.parameters,
			})
			if err != nil {
				span.RecordError(err)
				return &ExecutionResult{Status: core.RunStatusError}, err
			}

			acc := &Accumulator{}
			resp, err := acc.ReadStream(stream, cb)
			if err != nil {
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
			toolResults := []*responses.FunctionCallOutputMessage{}
			for _, toolCall := range runState.PendingToolCalls {
				tool := e.findTool(ctx, toolCall.Name)
				if tool == nil {
					slog.ErrorContext(ctx, "tool not found", slog.String("tool_name", toolCall.Name))
					continue
				}

				var toolResult *responses.FunctionCallOutputMessage
				if slices.Contains(rejectedToolCallIds, toolCall.CallID) {
					toolResult = &responses.FunctionCallOutputMessage{
						ID:     toolCall.ID,
						CallID: toolCall.CallID,
						Output: responses.FunctionCallOutputContentUnion{
							OfString: utils.Ptr("Request to call this tool has been declined"),
						},
					}
				} else {
					toolResult, err = tool.Execute(ctx, &toolCall)
					if err != nil {
						slog.ErrorContext(ctx, "tool call execution failed", slog.Any("error", err))
						return &ExecutionResult{Status: core.RunStatusError}, err
					}
				}

				toolResults = append(toolResults, toolResult)
				cb(&responses.ResponseChunk{
					OfFunctionCallOutput: toolResult,
				})

				// Add tool result to history
				e.history.AddMessages(ctx, []responses.InputMessageUnion{
					{
						OfFunctionCallOutput: toolResult,
					},
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
			// Save and exit - will resume later
			e.history.SaveMessages(ctx, runState.ToMeta(traceid))

			// Run paused
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

			return &ExecutionResult{
				Status:           core.RunStatusPaused,
				PendingApprovals: runState.PendingToolCalls,
			}, nil

		case core.StepComplete:
			// Save final state
			e.history.SaveMessages(ctx, runState.ToMeta(traceid))

			// Run end
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

			return &ExecutionResult{
				Status: core.RunStatusCompleted,
				Output: finalOutput,
			}, nil
		}
	}

	// Max loops exceeded
	return &ExecutionResult{Status: core.RunStatusError}, errors.New("max loops exceeded")
}

// partitionByApproval splits tool calls into those needing approval and those that can execute immediately
func (e *Agent) partitionByApproval(ctx context.Context, toolCalls []responses.FunctionCallMessage) (needsApproval []responses.FunctionCallMessage, immediate []responses.FunctionCallMessage) {
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
func (e *Agent) findTool(ctx context.Context, toolName string) core.Tool {
	for _, tool := range e.tools {
		if t := tool.Tool(ctx); t != nil && t.OfFunction != nil && t.OfFunction.Name == toolName {
			return tool
		}
	}
	return nil
}

type Accumulator struct {
}

func (a *Accumulator) ReadStream(stream chan *responses.ResponseChunk, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error) {
	// Process stream
	finalOutput := []responses.OutputMessageUnion{}
	var usage *responses.Usage
	for chunk := range stream {
		cb(chunk)
		switch chunk.ChunkType() {
		case "response.output_item.done":
			if chunk.OfOutputItemDone.Item.Type == "message" {
				for _, content := range chunk.OfOutputItemDone.Item.Content {
					if content.OfOutputText != nil {
						finalOutput = append(finalOutput, responses.OutputMessageUnion{
							OfOutputMessage: &responses.OutputMessage{
								ID:   chunk.OfOutputItemDone.Item.Id,
								Role: constants.RoleAssistant,
								Content: responses.OutputContent{
									content,
								},
							},
						})
					}
				}
			}

			if chunk.OfOutputItemDone.Item.Type == "reasoning" {
				var encryptedContent *string
				if chunk.OfOutputItemDone.Item.EncryptedContent != nil {
					encryptedContent = chunk.OfOutputItemDone.Item.EncryptedContent
				}

				finalOutput = append(finalOutput, responses.OutputMessageUnion{
					OfReasoning: &responses.ReasoningMessage{
						ID:               chunk.OfOutputItemDone.Item.Id,
						Summary:          chunk.OfOutputItemDone.Item.Summary,
						EncryptedContent: encryptedContent,
					},
				})
			}

			if chunk.OfOutputItemDone.Item.Type == "function_call" {
				finalOutput = append(finalOutput, responses.OutputMessageUnion{
					OfFunctionCall: &responses.FunctionCallMessage{
						ID:        chunk.OfOutputItemDone.Item.Id,
						CallID:    *chunk.OfOutputItemDone.Item.CallID,
						Name:      *chunk.OfOutputItemDone.Item.Name,
						Arguments: *chunk.OfOutputItemDone.Item.Arguments,
					},
				})
			}

			if chunk.OfOutputItemDone.Item.Type == "image_generation_call" {
				finalOutput = append(finalOutput, responses.OutputMessageUnion{
					OfImageGenerationCall: &responses.ImageGenerationCallMessage{
						ID:           chunk.OfOutputItemDone.Item.Id,
						Status:       chunk.OfOutputItemDone.Item.Status,
						Background:   *chunk.OfOutputItemDone.Item.Background,
						OutputFormat: *chunk.OfOutputItemDone.Item.OutputFormat,
						Quality:      *chunk.OfOutputItemDone.Item.Quality,
						Size:         *chunk.OfOutputItemDone.Item.Size,
						Result:       *chunk.OfOutputItemDone.Item.Result,
					},
				})
			}

		case "response.completed":
			usage = &chunk.OfResponseCompleted.Response.Usage
		}
	}

	return &responses.Response{
		Output: finalOutput,
		Usage:  usage,
	}, nil
}
