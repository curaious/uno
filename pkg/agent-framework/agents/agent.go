package agents

import (
	"context"
	"log/slog"

	"github.com/bytedance/sonic"
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
	name                string
	output              map[string]any
	history             core.ChatHistory
	instruction         string
	instructionProvider core.SystemPromptProvider
	tools               []core.Tool
	llm                 llm.Provider
	parameters          responses.Parameters
}

type AgentOptions struct {
	History             core.ChatHistory
	Instruction         string
	InstructionProvider core.SystemPromptProvider
	Parameters          responses.Parameters

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
		name:                opts.Name,
		output:              opts.Output,
		history:             opts.History,
		instruction:         opts.Instruction,
		instructionProvider: opts.InstructionProvider,
		tools:               opts.Tools,
		llm:                 opts.LLM,
		parameters:          opts.Parameters,
	}
}

func (e *Agent) Execute(ctx context.Context, msgs []responses.InputMessageUnion, cb func(*responses.ResponseChunk)) ([]responses.OutputMessageUnion, error) {
	ctx, span := tracer.Start(ctx, "Agent.Execute")
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

	// Agent executor loop
	for {
		// Get the messages from the conversation history
		convMessages, err := e.history.GetMessages(ctx)
		if err != nil {
			span.RecordError(err)
			return finalOutput, err
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
			return finalOutput, err
		}

		acc := &Accumulator{}
		resp, err := acc.ReadStream(stream, cb)
		if err != nil {
			span.RecordError(err)
			return finalOutput, err
		}

		finalOutput = append(finalOutput, resp.Output...)

		// Track the LLM's usage
		runUsage.InputTokens += resp.Usage.InputTokens
		runUsage.OutputTokens += resp.Usage.OutputTokens
		runUsage.InputTokensDetails.CachedTokens += resp.Usage.InputTokensDetails.CachedTokens
		runUsage.TotalTokens += resp.Usage.TotalTokens

		// Execute tool calls
		toolResults := []*responses.FunctionCallOutputMessage{}
		hasToolCalls := false

		// Extract and handle tool calls
		for _, msg := range resp.Output {
			if msg.OfFunctionCall == nil {
				continue
			}

			hasToolCalls = true

			args := map[string]interface{}{}
			if err := sonic.Unmarshal([]byte(msg.OfFunctionCall.Arguments), &args); err != nil {
				return finalOutput, err
			}

			for _, tool := range e.tools {
				if msg.OfFunctionCall.Name == tool.Tool(ctx).OfFunction.Name {
					toolResult, err := tool.Execute(ctx, msg.OfFunctionCall)
					if err != nil {
						slog.ErrorContext(ctx, "tool call execution failed", slog.Any("error", err))
						return finalOutput, err
					}

					toolResults = append(toolResults, toolResult)
					cb(&responses.ResponseChunk{
						OfFunctionCallOutput: toolResult,
					})
					break
				}
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
