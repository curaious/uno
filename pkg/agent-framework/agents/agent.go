package agents

import (
	"context"

	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/constants"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type Agent struct {
	name        string
	output      map[string]any
	history     core.ChatHistory
	instruction core.SystemPromptProvider
	tools       []core.Tool
	llm         llm.Provider
	parameters  responses.Parameters
	runtime     AgentRuntime
}

type AgentOptions struct {
	History     core.ChatHistory
	Instruction core.SystemPromptProvider
	Parameters  responses.Parameters

	Name    string
	LLM     llm.Provider
	Output  map[string]any
	Tools   []core.Tool
	Runtime AgentRuntime
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
		runtime:     opts.Runtime,
	}
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
	Status           core.RunStatus
	Output           []responses.OutputMessageUnion
	PendingApprovals []responses.FunctionCallMessage
}

func (e *Agent) Execute(ctx context.Context, in *AgentInput) (*AgentOutput, error) {
	// Delegate to runtime, or use default LocalRuntime if none is set
	runtime := e.runtime
	if runtime == nil {
		runtime = DefaultRuntime()
	}
	return runtime.Run(ctx, e, in)
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
