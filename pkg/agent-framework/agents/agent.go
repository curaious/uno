package agents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/curaious/uno/internal/services/conversation"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/constants"
	"github.com/curaious/uno/pkg/llm/responses"
	internal_adapters "github.com/curaious/uno/pkg/sdk/adapters"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	tracer = otel.Tracer("Agent")
)

type AgentRuntime interface {
	Run(ctx context.Context, agent *Agent, in *AgentInput) (*AgentOutput, error)
}

type DurableExecutor interface {
	history.ConversationPersistenceAdapter
	core.SystemPromptProvider
	NewStreamingResponses(ctx context.Context, in *responses.Request) (*responses.Response, error)
	CallTool(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error)
}

type Agent struct {
	name        string
	output      map[string]any
	history     *history.CommonConversationManager
	instruction core.SystemPromptProvider
	tools       []core.Tool
	mcpServers  []*mcpclient.MCPClient
	llm         llm.Provider
	parameters  responses.Parameters
	runtime     AgentRuntime
	maxLoops    int
}

type AgentOptions struct {
	History     *history.CommonConversationManager
	Instruction core.SystemPromptProvider
	Parameters  responses.Parameters

	Name       string
	LLM        llm.Provider
	Output     map[string]any
	Tools      []core.Tool
	McpServers []*mcpclient.MCPClient
	Runtime    AgentRuntime
	MaxLoops   int
}

func NewAgent(opts *AgentOptions) *Agent {
	maxLoops := opts.MaxLoops
	if maxLoops <= 0 {
		maxLoops = 50
	}

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

	if opts.History == nil {
		opts.History = history.NewConversationManager(internal_adapters.NewInMemoryConversationPersistence())
	}

	return &Agent{
		name:        opts.Name,
		output:      opts.Output,
		history:     opts.History,
		instruction: opts.Instruction,
		tools:       opts.Tools,
		mcpServers:  opts.McpServers,
		llm:         opts.LLM,
		parameters:  opts.Parameters,
		runtime:     opts.Runtime,
		maxLoops:    maxLoops,
	}
}

func (e *Agent) Name() string {
	return e.name
}

func (e *Agent) PrepareMCPTools(ctx context.Context, runContext map[string]any) ([]core.Tool, error) {
	coreTools := []core.Tool{}
	if e.mcpServers != nil {
		for _, mcpServer := range e.mcpServers {
			cli, err := mcpServer.GetClient(ctx, runContext)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize MCP server: %w", err)
			}
			coreTools = append(coreTools, cli.GetTools()...)
		}
	}

	return coreTools, nil
}

func (e *Agent) LoadMessages(ctx context.Context, namespace string, previousMessageID string) ([]conversation.ConversationMessage, error) {
	return e.history.ConversationPersistenceAdapter.LoadMessages(ctx, namespace, previousMessageID)
}

func (e *Agent) SaveMessages(ctx context.Context, namespace string, msgId string, previousMsgId string, conversationId string, messages []responses.InputMessageUnion, meta map[string]any) error {
	return e.history.ConversationPersistenceAdapter.SaveMessages(ctx, namespace, msgId, previousMsgId, conversationId, messages, meta)
}

func (e *Agent) SaveSummary(ctx context.Context, namespace string, summary conversation.Summary) error {
	return e.history.ConversationPersistenceAdapter.SaveSummary(ctx, namespace, summary)
}

func (e *Agent) GetPrompt(ctx context.Context, runContext map[string]any) (string, error) {
	return e.instruction.GetPrompt(ctx, runContext)
}

func (e *Agent) NewStreamingResponses(ctx context.Context, req *responses.Request) (*responses.Response, error) {
	stream, err := e.llm.NewStreamingResponses(ctx, req)
	if err != nil {
		return nil, err
	}

	acc := Accumulator{}
	return acc.ReadStream(stream, core.NilCallback)
}

func (e *Agent) CallTool(ctx context.Context, toolCall *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	tool := findTool(ctx, e.tools, toolCall.Name)
	if tool == nil {
		return nil, errors.New("tool not found: " + toolCall.Name)
	}

	return tool.Execute(ctx, toolCall)
}

type AgentInput struct {
	Namespace         string
	PreviousMessageID string
	Messages          []responses.InputMessageUnion
	RunContext        map[string]any
	Callback          func(chunk *responses.ResponseChunk)
}

// AgentOutput represents the result of agent execution
type AgentOutput struct {
	RunID            string                          `json:"run_id"`
	Status           core.RunStatus                  `json:"status"`
	Output           []responses.InputMessageUnion   `json:"output"`
	PendingApprovals []responses.FunctionCallMessage `json:"pending_approvals"`
}

func (e *Agent) Execute(ctx context.Context, in *AgentInput) (*AgentOutput, error) {
	// Delegate to runtime, or use default LocalRuntime if none is set
	runtime := e.runtime
	if runtime != nil {
		return runtime.Run(ctx, e, in)
	}

	return e.ExecuteWithExecutor(ctx, in, e)
}

func (e *Agent) ExecuteWithExecutor(ctx context.Context, in *AgentInput, executor DurableExecutor) (*AgentOutput, error) {
	ctx, span := tracer.Start(ctx, "Agent.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("agent.name", e.name))

	/* Prepare MCP Tools Activity */
	mcpTools, err := e.PrepareMCPTools(ctx, in.RunContext)
	if err != nil {
		return nil, err
	}
	/* End Activity */

	tools := append(e.tools, mcpTools...)

	toolDefs := make([]responses.ToolUnion, len(tools))
	for idx, coreTool := range tools {
		toolDefs[idx] = *coreTool.Tool(ctx)
	}

	cb := in.Callback
	if cb == nil {
		cb = core.NilCallback
	}

	var runId string
	chatHistory := history.NewRun(executor, e.history.Options...)

	_, err = chatHistory.LoadMessages(ctx, in.Namespace, in.PreviousMessageID)
	if err != nil {
		span.RecordError(err)
		return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
	}

	// Load run state from meta (in-memory, no DB call)
	runId = chatHistory.GetMessageID()
	meta := chatHistory.GetMeta()
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
		if len(in.Messages) > 0 {
			chatHistory.AddMessages(ctx, in.Messages, nil)
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
			// Expected an approval response message
			if len(in.Messages) == 0 || in.Messages[0].OfFunctionCallApprovalResponse == nil {
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, errors.New("expected approval response message")
			}
			runState.CurrentStep = core.StepExecuteTools
			rejectedToolCallIds = in.Messages[0].OfFunctionCallApprovalResponse.RejectedCallIds
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

	// DURABLE CHECKPOINT: Load system instruction (potential DB read)
	instruction := "You are a helpful assistant."
	if e.instruction != nil {
		instruction, err = executor.GetPrompt(ctx, in.RunContext)
		if err != nil {
			span.RecordError(err)
			return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
		}
	}

	// Apply structured output format if configured
	parameters := e.parameters
	if e.output != nil {
		format := map[string]any{
			"type":   "json_schema",
			"name":   "structured_output",
			"strict": false,
			"schema": e.output,
		}
		parameters.Text = &responses.TextFormat{
			Format: format,
		}
	}

	finalOutput := []responses.InputMessageUnion{}

	// Main loop - driven by state machine
	for runState.LoopIteration < e.maxLoops {
		switch runState.NextStep() {

		case core.StepCallLLM:
			// TODO: make `GetMessages` as durable step
			convMessages, err := chatHistory.GetMessages(ctx)
			if err != nil {
				span.RecordError(err)
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

			resp, err := executor.NewStreamingResponses(ctx, &responses.Request{
				Instructions: utils.Ptr(instruction),
				Input: responses.InputUnion{
					OfInputMessageList: convMessages,
				},
				Tools:      toolDefs,
				Parameters: parameters,
			})
			if err != nil {
				span.RecordError(err)
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

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
					return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
				}
				inputMsgs = append(inputMsgs, inputMsg)
			}

			chatHistory.AddMessages(ctx, inputMsgs, resp.Usage)
			finalOutput = append(finalOutput, inputMsgs...)

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
				needsApproval, immediate := partitionByApproval(ctx, tools, toolCalls)

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
				tool := findTool(ctx, tools, toolCall.Name)
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
					toolResult, err = executor.CallTool(ctx, &toolCall)
					if err != nil {
						span.RecordError(err)
						return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
					}
				}

				cb(&responses.ResponseChunk{
					OfFunctionCallOutput: toolResult,
				})

				toolResultMsg := []responses.InputMessageUnion{
					{OfFunctionCallOutput: toolResult},
				}

				// Add tool result to history
				chatHistory.AddMessages(ctx, toolResultMsg, nil)
				finalOutput = append(finalOutput, toolResultMsg...)
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
			err = chatHistory.SaveMessages(ctx, runState.ToMeta(traceid))
			if err != nil {
				span.RecordError(err)
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

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

			return &AgentOutput{
				RunID:            runId,
				Status:           core.RunStatusPaused,
				PendingApprovals: runState.PendingToolCalls,
			}, nil

		case core.StepComplete:
			// DURABLE CHECKPOINT: Save final state (DB write)
			err = chatHistory.SaveMessages(ctx, runState.ToMeta(traceid))
			if err != nil {
				span.RecordError(err)
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

			// DURABLE: Emit run completed event (side effect)
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

			return &AgentOutput{
				RunID:  runId,
				Status: core.RunStatusCompleted,
				Output: finalOutput,
			}, nil
		}
	}

	// Max loops exceeded
	return &AgentOutput{Status: core.RunStatusError, RunID: runId}, fmt.Errorf("exceeded maximum loops (%d)", e.maxLoops)
}

// partitionByApproval splits tool calls into those needing approval and those that can execute immediately
func partitionByApproval(ctx context.Context, tools []core.Tool, toolCalls []responses.FunctionCallMessage) (needsApproval []responses.FunctionCallMessage, immediate []responses.FunctionCallMessage) {
	for _, toolCall := range toolCalls {
		tool := findTool(ctx, tools, toolCall.Name)
		if tool != nil && tool.NeedApproval() {
			needsApproval = append(needsApproval, toolCall)
		} else {
			immediate = append(immediate, toolCall)
		}
	}
	return needsApproval, immediate
}

// findTool finds a tool by name
func findTool(ctx context.Context, tools []core.Tool, toolName string) core.Tool {
	for _, tool := range tools {
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
						ID:               chunk.OfOutputItemDone.Item.Id,
						CallID:           *chunk.OfOutputItemDone.Item.CallID,
						Name:             *chunk.OfOutputItemDone.Item.Name,
						Arguments:        *chunk.OfOutputItemDone.Item.Arguments,
						ThoughtSignature: chunk.OfOutputItemDone.Item.ThoughtSignature,
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
