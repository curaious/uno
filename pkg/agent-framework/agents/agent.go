package agents

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/constants"
	"github.com/curaious/uno/pkg/llm/responses"
	internal_adapters "github.com/curaious/uno/pkg/sdk/adapters"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("Agent")
)

type MCPToolset interface {
	GetName() string
	ListTools(ctx context.Context, runContext map[string]any) ([]core.Tool, error)
}

type LLM interface {
	NewStreamingResponses(ctx context.Context, in *responses.Request, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error)
}

type WrappedLLM struct {
	llm llm.Provider
}

func (l *WrappedLLM) NewStreamingResponses(ctx context.Context, in *responses.Request, cb func(chunk *responses.ResponseChunk)) (*responses.Response, error) {
	acc := Accumulator{}

	stream, err := l.llm.NewStreamingResponses(ctx, in)
	if err != nil {
		return nil, err
	}

	return acc.ReadStream(stream, cb)
}

type AgentRuntime interface {
	Run(ctx context.Context, agent *Agent, in *AgentInput) (*AgentOutput, error)
}

type Agent struct {
	Name         string
	output       map[string]any
	history      *history.CommonConversationManager
	instruction  core.SystemPromptProvider
	tools        []core.Tool
	mcpServers   []MCPToolset
	llm          LLM
	parameters   responses.Parameters
	runtime      AgentRuntime
	maxLoops     int
	streamBroker core.StreamBroker
}

type AgentOptions struct {
	History     *history.CommonConversationManager
	Instruction core.SystemPromptProvider
	Parameters  responses.Parameters

	Name       string
	LLM        llm.Provider
	Output     map[string]any
	Tools      []core.Tool
	McpServers []MCPToolset
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
		Name:        opts.Name,
		output:      opts.Output,
		history:     opts.History,
		instruction: opts.Instruction,
		tools:       opts.Tools,
		mcpServers:  opts.McpServers,
		llm:         &WrappedLLM{opts.LLM},
		parameters:  opts.Parameters,
		runtime:     opts.Runtime,
		maxLoops:    maxLoops,
	}
}

func (e *Agent) WithLLM(wrappedLLM LLM) *Agent {
	return &Agent{
		Name:         e.Name,
		output:       e.output,
		history:      e.history,
		instruction:  e.instruction,
		tools:        e.tools,
		mcpServers:   e.mcpServers,
		llm:          wrappedLLM,
		parameters:   e.parameters,
		runtime:      e.runtime,
		maxLoops:     e.maxLoops,
		streamBroker: e.streamBroker,
	}
}

func (e *Agent) PrepareMCPTools(ctx context.Context, runContext map[string]any) ([]core.Tool, error) {
	coreTools := []core.Tool{}
	if e.mcpServers != nil {
		for _, mcpServer := range e.mcpServers {
			mcpTools, err := mcpServer.ListTools(ctx, runContext)
			if err != nil {
				return nil, fmt.Errorf("failed to list MCP tools: %w", err)
			}

			coreTools = append(coreTools, mcpTools...)
		}
	}

	return coreTools, nil
}

func (e *Agent) GetRunID(ctx context.Context) string {
	return uuid.NewString()
}

// StartSpan creates a real OTel span for local (non-durable) execution.
func (e *Agent) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	ctx, span := tracer.Start(ctx, name)
	span.SetAttributes(attrs...)
	return ctx, func() { span.End() }
}

type AgentInput struct {
	Namespace         string                               `json:"namespace"`
	PreviousMessageID string                               `json:"previous_message_id"`
	Messages          []responses.InputMessageUnion        `json:"messages"`
	RunContext        map[string]any                       `json:"run_context"`
	Callback          func(chunk *responses.ResponseChunk) `json:"-"`
	StreamBroker      core.StreamBroker                    `json:"-"`
}

// AgentOutput represents the result of agent execution
type AgentOutput struct {
	RunID            string                          `json:"run_id"`
	Status           core.RunStatus                  `json:"status"`
	Output           []responses.InputMessageUnion   `json:"output"`
	PendingApprovals []responses.FunctionCallMessage `json:"pending_approvals"`
}

func (e *Agent) Execute(ctx context.Context, in *AgentInput) (*AgentOutput, error) {
	if in.Callback == nil {
		in.Callback = core.NilCallback
	}

	// Delegate to runtime, or use default LocalRuntime if none is set
	runtime := e.runtime
	if runtime != nil {
		return runtime.Run(ctx, e, in)
	}

	return e.ExecuteWithExecutor(ctx, in, in.Callback)
}

func (e *Agent) ExecuteWithExecutor(ctx context.Context, in *AgentInput, cb func(chunk *responses.ResponseChunk)) (*AgentOutput, error) {
	// Connect to MCP servers, and list the tools
	mcpTools, err := e.PrepareMCPTools(ctx, in.RunContext)
	if err != nil {
		return nil, err
	}

	// Merge MCP tools with other tools
	tools := append(e.tools, mcpTools...)

	// Create tool schemas for input payload
	var toolDefs []responses.ToolUnion
	if len(tools) > 0 {
		toolDefs = make([]responses.ToolUnion, len(tools))
		for idx, coreTool := range tools {
			toolDefs[idx] = *coreTool.Tool(ctx)
		}
	}

	// Generate a run ID
	run, err := history.NewRun(ctx, e.history.ConversationPersistenceAdapter, in.Namespace, in.PreviousMessageID, in.Messages)
	if err != nil {
		return &AgentOutput{Status: core.RunStatusError, RunID: ""}, err
	}

	// Load run state from meta (in-memory, no DB call)
	runId := run.GetMessageID()

	// TODO: what's the implication of obtaining traceid from context in case of durable execution?
	var traceid string
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		traceid = sc.TraceID().String()
	}

	// Collect tool rejections
	var rejectedToolCallIds []string
	if run.RunState.IsPaused() {
		if run.RunState.CurrentStep == core.StepAwaitApproval {
			rejectedToolCallIds = in.Messages[0].OfFunctionCallApprovalResponse.RejectedCallIds
		}
	}

	// Emit run.created
	// TODO: make this a durable step to avoid resending on replays
	e.runCreated(ctx, runId, traceid, cb)

	// Get the prompt
	instruction := "You are a helpful assistant."
	if e.instruction != nil {
		instruction, err = e.instruction.GetPrompt(ctx, in.RunContext)
		if err != nil {
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
	for run.RunState.LoopIteration < e.maxLoops {
		switch run.RunState.NextStep() {

		case core.StepCallLLM:
			// TODO: make `GetMessages` as durable step, to avoid summarisation on replays
			convMessages, err := run.GetMessages(ctx)
			if err != nil {
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

			resp, err := e.llm.NewStreamingResponses(ctx, &responses.Request{
				Instructions: utils.Ptr(instruction),
				Input: responses.InputUnion{
					OfInputMessageList: convMessages,
				},
				Tools:      toolDefs,
				Parameters: parameters,
			}, cb)
			if err != nil {
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

			// Track the LLM's usage
			run.TrackUsage(resp.Usage)

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

			run.AddMessages(ctx, inputMsgs, resp.Usage)
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
				run.RunState.TransitionToComplete()
			} else {
				// Partition tools by approval requirement
				needsApproval, immediate := partitionByApproval(ctx, tools, toolCalls)

				// Execute immediate tools first (if any), then handle approval
				if len(immediate) > 0 {
					run.RunState.TransitionToExecuteTools(immediate)
					// Store tools needing approval for after immediate execution
					if len(needsApproval) > 0 {
						run.RunState.ToolsAwaitingApproval = needsApproval
					}
				} else if len(needsApproval) > 0 {
					// Only approval-required tools, no immediate ones
					run.RunState.TransitionToAwaitApproval(needsApproval)
				}
			}

		case core.StepExecuteTools:
			// Execute pending tool calls
			for _, toolCall := range run.RunState.PendingToolCalls {
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
					toolResult, err = tool.Execute(ctx, &toolCall)
					if err != nil {
						return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
					}
				}

				// TODO: Make this a durable step to avoid resending
				cb(&responses.ResponseChunk{
					OfFunctionCallOutput: toolResult,
				})

				toolResultMsg := []responses.InputMessageUnion{
					{OfFunctionCallOutput: toolResult},
				}

				// Add tool result to history
				run.AddMessages(ctx, toolResultMsg, nil)
				finalOutput = append(finalOutput, toolResultMsg...)
			}

			run.RunState.ClearPendingTools()

			// Check if there are tools waiting for approval (queued during immediate execution)
			if run.RunState.HasToolsAwaitingApproval() {
				run.RunState.PromoteAwaitingToApproval()
			} else {
				run.RunState.TransitionToLLM()
			}

		case core.StepAwaitApproval:
			err = run.SaveMessages(ctx, run.RunState.ToMeta(traceid))
			if err != nil {
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

			// TODO: make this a durable step to avoid resending on replays
			e.runPaused(ctx, runId, traceid, run.RunState, cb)

			return &AgentOutput{
				RunID:            runId,
				Status:           core.RunStatusPaused,
				PendingApprovals: run.RunState.PendingToolCalls,
			}, nil

		case core.StepComplete:
			err = run.SaveMessages(ctx, run.RunState.ToMeta(traceid))
			if err != nil {
				return &AgentOutput{Status: core.RunStatusError, RunID: runId}, err
			}

			// TODO: make this a durable step to avoid resending on replays
			e.runCompleted(ctx, runId, traceid, run.RunState, cb)

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

func (e *Agent) runCreated(ctx context.Context, runId string, traceId string, cb func(chunk *responses.ResponseChunk)) error {
	cb(&responses.ResponseChunk{
		OfRunCreated: &responses.ChunkRun[constants.ChunkTypeRunCreated]{
			RunState: responses.ChunkRunData{
				Id:      runId,
				Object:  "run",
				Status:  "created",
				TraceID: traceId,
			},
		},
	})

	cb(&responses.ResponseChunk{
		OfRunInProgress: &responses.ChunkRun[constants.ChunkTypeRunInProgress]{
			RunState: responses.ChunkRunData{
				Id:      runId,
				Object:  "run",
				Status:  "in_progress",
				TraceID: traceId,
			},
		},
	})

	return nil
}

func (e *Agent) runPaused(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	cb(&responses.ResponseChunk{
		OfRunPaused: &responses.ChunkRun[constants.ChunkTypeRunPaused]{
			RunState: responses.ChunkRunData{
				Id:               runId,
				Object:           "run",
				Status:           "paused",
				PendingToolCalls: runState.PendingToolCalls,
				Usage:            runState.Usage,
				TraceID:          traceId,
			},
		},
	})

	return nil
}

func (e *Agent) runCompleted(ctx context.Context, runId string, traceId string, runState *core.RunState, cb func(chunk *responses.ResponseChunk)) error {
	cb(&responses.ResponseChunk{
		OfRunCompleted: &responses.ChunkRun[constants.ChunkTypeRunCompleted]{
			RunState: responses.ChunkRunData{
				Id:      runId,
				Object:  "run",
				Status:  "completed",
				Usage:   runState.Usage,
				TraceID: traceId,
			},
		},
	})
	return nil
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
