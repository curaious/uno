package anthropic_responses

import (
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/llm/constants"
	"github.com/curaious/uno/pkg/llm/responses"
)

func NativeRequestToRequest(in *responses.Request) *Request {
	if in.MaxOutputTokens == nil {
		in.MaxOutputTokens = utils.Ptr(512)
	}

	if in.MaxToolCalls != nil {
		slog.Warn("max tool call is not supported for anthropic models")
	}

	if in.ParallelToolCalls != nil {
		slog.Warn("parallel tool call is not supported for anthropic models")
	}

	out := &Request{
		Temperature: in.Temperature,
		MaxTokens:   *in.MaxOutputTokens,
		TopP:        in.TopP,
		TopK:        in.TopLogprobs,
		Model:       in.Model,
		Messages:    NativeMessagesToMessage(in.Input),
		Metadata:    in.Metadata,
		Tools:       NativeToolsToTools(in.Tools),
		Stream:      in.Stream,
	}

	if in.Instructions != nil {
		out.System = []TextContent{
			{
				Text: *in.Instructions,
			},
		}
	}

	if in.Reasoning != nil && in.Reasoning.BudgetTokens != nil {
		if *in.Reasoning.BudgetTokens >= 1024 && *in.Reasoning.BudgetTokens < *in.MaxOutputTokens {
			out.Thinking = &ThinkingParam{
				Type:         utils.Ptr("enabled"),
				BudgetTokens: in.Reasoning.BudgetTokens,
			}
		}
	}

	if in.Text != nil {
		out.OutputFormat = in.Text.Format

		// Since anthropic doesn't allow extra keys, we delete keys that are specific to openai
		delete(out.OutputFormat, "name")
		delete(out.OutputFormat, "strict")
	}

	return out
}

func NativeRoleToRole(in constants.Role) Role {
	switch in {
	case constants.RoleUser:
		return RoleUser
	case constants.RoleSystem, constants.RoleDeveloper:
		return RoleUser
	case constants.RoleAssistant:
		return RoleAssistant
	}

	return RoleAssistant
}

func NativeToolsToTools(nativeTools []responses.ToolUnion) []ToolUnion {
	out := []ToolUnion{}

	for _, nativeTool := range nativeTools {
		if nativeTool.OfFunction != nil {
			out = append(out, ToolUnion{
				OfCustomTool: &CustomTool{
					Type:        "custom",
					Name:        nativeTool.OfFunction.Name,
					Description: nativeTool.OfFunction.Description,
					InputSchema: nativeTool.OfFunction.Parameters,
				},
			})
		}
	}

	return out
}

func NativeMessagesToMessage(in responses.InputUnion) []MessageUnion {
	out := []MessageUnion{}

	if in.OfString != nil {
		out = append(out, MessageUnion{
			Role: RoleUser,
			Content: Contents{
				ContentUnion{
					OfText: &TextContent{
						Text: *in.OfString,
					},
				},
			},
		})
		return out
	}

	if in.OfInputMessageList != nil {
		for _, nativeMessage := range in.OfInputMessageList {
			if nativeMessage.OfEasyInput != nil {
				contents := Contents{}

				if nativeMessage.OfEasyInput.Content.OfString != nil {
					contents = append(contents, ContentUnion{OfText: &TextContent{
						Text: *nativeMessage.OfEasyInput.Content.OfString,
					}})
				}

				if nativeMessage.OfEasyInput.Content.OfInputMessageList != nil {
					for _, nativeContent := range nativeMessage.OfEasyInput.Content.OfInputMessageList {
						if nativeContent.OfInputText != nil {
							contents = append(contents, ContentUnion{
								OfText: &TextContent{
									Text: nativeContent.OfInputText.Text,
								},
							})
						}

						if nativeContent.OfOutputText != nil {
							contents = append(contents, ContentUnion{
								OfText: &TextContent{
									Text: nativeContent.OfOutputText.Text,
								},
							})
						}
					}
				}

				out = append(out, MessageUnion{
					Role:    NativeRoleToRole(nativeMessage.OfEasyInput.Role),
					Content: contents,
				})
			}

			if nativeMessage.OfInputMessage != nil {
				contents := Contents{}

				for _, nativeContent := range nativeMessage.OfInputMessage.Content {
					if nativeContent.OfInputText != nil {
						contents = append(contents, ContentUnion{
							OfText: &TextContent{
								Text: nativeContent.OfInputText.Text,
							},
						})
					}

					if nativeContent.OfOutputText != nil {
						contents = append(contents, ContentUnion{
							OfText: &TextContent{
								Text: nativeContent.OfOutputText.Text,
							},
						})
					}
				}

				out = append(out, MessageUnion{
					Role:    NativeRoleToRole(nativeMessage.OfInputMessage.Role),
					Content: contents,
				})
			}

			if nativeMessage.OfFunctionCall != nil {
				args := map[string]any{}
				if err := sonic.Unmarshal([]byte(nativeMessage.OfFunctionCall.Arguments), &args); err != nil {
					slog.Warn("unable to unmarshal tool_use args - string into map[string]any")
				}

				out = append(out, MessageUnion{
					Role: RoleAssistant,
					Content: Contents{
						{
							OfToolUse: &ToolUseContent{
								ID:    nativeMessage.OfFunctionCall.CallID,
								Name:  nativeMessage.OfFunctionCall.Name,
								Input: args,
							},
						},
					},
				})
			}

			if nativeMessage.OfFunctionCallOutput != nil {
				output := Contents{}

				if nativeMessage.OfFunctionCallOutput.Output.OfString != nil {
					output = append(output, ContentUnion{
						OfToolResult: &ToolUseResultContent{
							ToolUseID: nativeMessage.OfFunctionCallOutput.CallID,
							Content: []ContentUnion{
								{
									OfText: &TextContent{
										Text: *nativeMessage.OfFunctionCallOutput.Output.OfString,
									},
								},
							},
						},
					})
				}

				if nativeMessage.OfFunctionCallOutput.Output.OfList != nil {
					for _, nativeOutput := range nativeMessage.OfFunctionCallOutput.Output.OfList {
						if nativeOutput.OfInputText != nil {
							output = append(output, ContentUnion{
								OfToolResult: &ToolUseResultContent{
									ToolUseID: nativeMessage.OfFunctionCallOutput.CallID,
									Content: []ContentUnion{
										{
											OfText: &TextContent{
												Text: nativeOutput.OfInputText.Text,
											},
										},
									},
								},
							})
						}
					}
				}

				out = append(out, MessageUnion{
					Role:    RoleUser,
					Content: output,
				})
			}

			// Reasoning can be thinking or redacted_thinking
			if nativeMessage.OfReasoning != nil {
				if nativeMessage.OfReasoning.EncryptedContent == nil || *nativeMessage.OfReasoning.EncryptedContent == "" {
					continue
				}

				// Thinking
				if nativeMessage.OfReasoning.Summary != nil {
					thinking := ""
					for _, nativeThinkingContent := range nativeMessage.OfReasoning.Summary {
						thinking += nativeThinkingContent.Text
					}

					out = append(out, MessageUnion{
						Role: RoleAssistant,
						Content: Contents{
							{
								OfThinking: &ThinkingContent{
									Thinking:  thinking,
									Signature: *nativeMessage.OfReasoning.EncryptedContent,
								},
							},
						},
					})
				}

				// Redacted Thinking
				if nativeMessage.OfReasoning.Summary == nil {
					out = append(out, MessageUnion{
						Role: RoleAssistant,
						Content: Contents{
							{
								OfThinking: &ThinkingContent{
									Signature: *nativeMessage.OfReasoning.EncryptedContent,
								},
							},
						},
					})
				}
			}
		}
	}

	return out
}

func NativeResponseToResponse(in *responses.Response) *Response {
	contents := Contents{}

	for _, nativeOutput := range in.Output {
		if nativeOutput.OfOutputMessage != nil {
			for _, nativeContent := range nativeOutput.OfOutputMessage.Content {
				contents = append(contents, ContentUnion{
					OfText: &TextContent{
						Type: "text",
						Text: nativeContent.OfOutputText.Text,
					},
				})
			}
		}

		if nativeOutput.OfFunctionCall != nil {
			contents = append(contents, ContentUnion{
				OfToolUse: &ToolUseContent{
					ID:    nativeOutput.OfFunctionCall.ID,
					Name:  nativeOutput.OfFunctionCall.Name,
					Input: nativeOutput.OfFunctionCall.Arguments,
				},
			})
		}

		if nativeOutput.OfReasoning != nil {
			summaryText := ""
			for _, nativeSummaryContent := range nativeOutput.OfReasoning.Summary {
				summaryText += nativeSummaryContent.Text
			}

			if summaryText != "" {
				contents = append(contents, ContentUnion{
					OfThinking: &ThinkingContent{
						Thinking:  summaryText,
						Signature: *nativeOutput.OfReasoning.EncryptedContent,
					},
				})
			} else {
				contents = append(contents, ContentUnion{
					OfThinking: &ThinkingContent{
						Signature: *nativeOutput.OfReasoning.EncryptedContent,
					},
				})
			}
		}
	}

	var stopReason StopReason
	var stopSequence string
	if in.Metadata != nil {
		if val, ok := in.Metadata["stop_reason"]; ok {
			stopReason = val.(StopReason)
		}
		if val, ok := in.Metadata["stop_sequence"]; ok {
			stopSequence = val.(string)
		}
	}

	return &Response{
		Model:        in.Model,
		Id:           in.ID,
		Type:         "message",
		Role:         RoleAssistant,
		Content:      contents,
		StopReason:   stopReason,
		StopSequence: stopSequence,
		Usage: &ChunkMessageUsage{
			InputTokens:              in.Usage.InputTokens,
			CacheCreationInputTokens: in.Usage.InputTokensDetails.CachedTokens,
			CacheReadInputTokens:     in.Usage.InputTokensDetails.CachedTokens,
			OutputTokens:             in.Usage.OutputTokens,
			CacheCreation:            nil,
			ServiceTier:              "",
		},
		ServiceTier: in.ServiceTier,
		Error:       nil,
	}
}

// =============================================================================
// Native to ResponseChunk Conversion
// =============================================================================

// NativeResponseChunkToResponseChunkConverter converts native stream chunks to Anthropic format.
// This converter is stateless since native format contains all necessary information in each chunk.
type NativeResponseChunkToResponseChunkConverter struct {
	OfResponseCreated *responses.ChunkResponse[constants.ChunkTypeResponseCreated]
}

// NativeResponseChunkToResponseChunk converts a single native chunk to zero or more Anthropic chunks.
func (c *NativeResponseChunkToResponseChunkConverter) NativeResponseChunkToResponseChunk(in *responses.ResponseChunk) []ResponseChunk {
	if in == nil {
		return nil
	}

	switch {
	case in.OfResponseCreated != nil:
		return c.handleResponseCreated(in.OfResponseCreated)
	case in.OfResponseInProgress != nil:
		return nil // No Anthropic equivalent
	case in.OfOutputItemAdded != nil:
		return c.handleOutputItemAdded(in.OfOutputItemAdded)
	case in.OfContentPartAdded != nil:
		return c.handleContentPartAdded(in.OfContentPartAdded)
	case in.OfOutputTextDelta != nil:
		return c.handleOutputTextDelta(in.OfOutputTextDelta)
	case in.OfOutputTextDone != nil:
		return nil // No Anthropic equivalent (block stop handles this)
	case in.OfContentPartDone != nil:
		return nil // No Anthropic equivalent
	case in.OfFunctionCallArgumentsDelta != nil:
		return c.handleFunctionCallArgumentsDelta(in.OfFunctionCallArgumentsDelta)
	case in.OfFunctionCallArgumentsDone != nil:
		return nil // No Anthropic equivalent
	case in.OfReasoningSummaryPartAdded != nil:
		return c.handleReasoningSummaryPartAdded(in.OfReasoningSummaryPartAdded)
	case in.OfReasoningSummaryTextDelta != nil:
		return c.handleReasoningSummaryTextDelta(in.OfReasoningSummaryTextDelta)
	case in.OfReasoningSummaryTextDone != nil:
		return nil // No Anthropic equivalent
	case in.OfReasoningSummaryPartDone != nil:
		return nil // No Anthropic equivalent
	case in.OfOutputItemDone != nil:
		return c.handleOutputItemDone(in.OfOutputItemDone)
	case in.OfResponseCompleted != nil:
		return c.handleResponseCompleted(in.OfResponseCompleted)
	}

	return nil
}

// =============================================================================
// Event Handlers
// =============================================================================

// handleResponseCreated emits message_start
func (c *NativeResponseChunkToResponseChunkConverter) handleResponseCreated(resp *responses.ChunkResponse[constants.ChunkTypeResponseCreated]) []ResponseChunk {
	c.OfResponseCreated = resp
	return []ResponseChunk{
		c.buildMessageStart(resp.Response.Id, resp.Response.Request.Model),
	}
}

// handleOutputItemAdded emits content_block_start for function_call only
// (text/message items defer to content_part.added since content type isn't known yet)
func (c *NativeResponseChunkToResponseChunkConverter) handleOutputItemAdded(item *responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]) []ResponseChunk {
	if item.Item.Type != "function_call" {
		return nil
	}
	return []ResponseChunk{
		c.buildContentBlockStartToolUse(item.OutputIndex, *item.Item.CallID, *item.Item.Name, item.Item.Arguments),
	}
}

// handleContentPartAdded emits content_block_start for text
func (c *NativeResponseChunkToResponseChunkConverter) handleContentPartAdded(part *responses.ChunkContentPart[constants.ChunkTypeContentPartAdded]) []ResponseChunk {
	if part.Part.OfOutputText == nil {
		return nil
	}
	return []ResponseChunk{
		c.buildContentBlockStartText(part.ContentIndex, part.Part.OfOutputText.Text),
	}
}

// handleOutputTextDelta emits content_block_delta with text_delta
func (c *NativeResponseChunkToResponseChunkConverter) handleOutputTextDelta(delta *responses.ChunkOutputText[constants.ChunkTypeOutputTextDelta]) []ResponseChunk {
	return []ResponseChunk{
		c.buildContentBlockDeltaText(delta.ContentIndex, delta.Delta),
	}
}

// handleFunctionCallArgumentsDelta emits content_block_delta with input_json_delta
func (c *NativeResponseChunkToResponseChunkConverter) handleFunctionCallArgumentsDelta(delta *responses.ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDelta]) []ResponseChunk {
	return []ResponseChunk{
		c.buildContentBlockDeltaInputJSON(delta.OutputIndex, delta.Arguments),
	}
}

// handleReasoningSummaryPartAdded emits content_block_start for thinking
func (c *NativeResponseChunkToResponseChunkConverter) handleReasoningSummaryPartAdded(part *responses.ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartAdded]) []ResponseChunk {
	return []ResponseChunk{
		c.buildContentBlockStartThinking(part.OutputIndex),
	}
}

// handleReasoningSummaryTextDelta emits content_block_delta with thinking or signature
func (c *NativeResponseChunkToResponseChunkConverter) handleReasoningSummaryTextDelta(delta *responses.ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDelta]) []ResponseChunk {
	if delta.EncryptedContent != nil {
		return []ResponseChunk{c.buildContentBlockDeltaSignature(delta.SummaryIndex, *delta.EncryptedContent)}
	}
	return []ResponseChunk{c.buildContentBlockDeltaThinking(delta.SummaryIndex, delta.Delta)}
}

// handleOutputItemDone emits content_block_stop
func (c *NativeResponseChunkToResponseChunkConverter) handleOutputItemDone(item *responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]) []ResponseChunk {
	return []ResponseChunk{
		c.buildContentBlockStop(item.OutputIndex),
	}
}

// handleResponseCompleted emits message_delta and message_stop
func (c *NativeResponseChunkToResponseChunkConverter) handleResponseCompleted(resp *responses.ChunkResponse[constants.ChunkTypeResponseCompleted]) []ResponseChunk {
	stopReason := "end_turn"
	if resp.Response.Status == "incomplete" {
		stopReason = "max_tokens"
	}

	return []ResponseChunk{
		c.buildMessageDelta(stopReason, resp.Response.Usage),
		c.buildMessageStop(),
	}
}

// =============================================================================
// Chunk Builders
// =============================================================================

func (c *NativeResponseChunkToResponseChunkConverter) buildMessageStart(id, model string) ResponseChunk {
	return ResponseChunk{
		OfMessageStart: &ChunkMessage[ChunkTypeMessageStart]{
			Type: ChunkTypeMessageStart("message_start"),
			Message: &ChunkMessageData{
				Id:      id,
				Type:    "message",
				Role:    "assistant",
				Model:   model,
				Content: []interface{}{},
				Usage:   &ChunkMessageUsage{InputTokens: 0, OutputTokens: 0},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockStartText(index int, text string) ResponseChunk {
	return ResponseChunk{
		OfContentBlockStart: &ChunkContentBlock[ChunkTypeContentBlockStart]{
			Type:         ChunkTypeContentBlockStart("content_block_start"),
			Index:        index,
			ContentBlock: &ContentUnion{OfText: &TextContent{Text: text}},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockStartToolUse(index int, callID, name string, args any) ResponseChunk {
	return ResponseChunk{
		OfContentBlockStart: &ChunkContentBlock[ChunkTypeContentBlockStart]{
			Type:  ChunkTypeContentBlockStart("content_block_start"),
			Index: index,
			ContentBlock: &ContentUnion{
				OfToolUse: &ToolUseContent{ID: callID, Name: name, Input: args},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockStartThinking(index int) ResponseChunk {
	return ResponseChunk{
		OfContentBlockStart: &ChunkContentBlock[ChunkTypeContentBlockStart]{
			Type:  ChunkTypeContentBlockStart("content_block_start"),
			Index: index,
			ContentBlock: &ContentUnion{
				OfThinking: &ThinkingContent{Thinking: "", Signature: ""},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockDeltaText(index int, text string) ResponseChunk {
	return ResponseChunk{
		OfContentBlockDelta: &ChunkContentBlock[ChunkTypeContentBlockDelta]{
			Type:  ChunkTypeContentBlockDelta("content_block_delta"),
			Index: index,
			Delta: &ChunkContentBlockDeltaUnion{
				OfText: &DeltaTextContent{Type: "text_delta", Text: text},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockDeltaInputJSON(index int, json string) ResponseChunk {
	return ResponseChunk{
		OfContentBlockDelta: &ChunkContentBlock[ChunkTypeContentBlockDelta]{
			Type:  ChunkTypeContentBlockDelta("content_block_delta"),
			Index: index,
			Delta: &ChunkContentBlockDeltaUnion{
				OfInputJSON: &DeltaInputJSONContent{Type: "input_json_delta", PartialJSON: json},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockDeltaThinking(index int, thinking string) ResponseChunk {
	return ResponseChunk{
		OfContentBlockDelta: &ChunkContentBlock[ChunkTypeContentBlockDelta]{
			Type:  ChunkTypeContentBlockDelta("content_block_delta"),
			Index: index,
			Delta: &ChunkContentBlockDeltaUnion{
				OfThinking: &DeltaThinkingContent{Thinking: thinking},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockDeltaSignature(index int, sig string) ResponseChunk {
	return ResponseChunk{
		OfContentBlockDelta: &ChunkContentBlock[ChunkTypeContentBlockDelta]{
			Type:  ChunkTypeContentBlockDelta("content_block_delta"),
			Index: index,
			Delta: &ChunkContentBlockDeltaUnion{
				OfThinkingSignature: &DeltaThinkingSignatureContent{Signature: sig},
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildContentBlockStop(index int) ResponseChunk {
	return ResponseChunk{
		OfContentBlockStop: &ChunkContentBlock[ChunkTypeContentBlockStop]{
			Type:  ChunkTypeContentBlockStop("content_block_stop"),
			Index: index,
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildMessageDelta(stopReason string, usage responses.Usage) ResponseChunk {
	return ResponseChunk{
		OfMessageDelta: &ChunkMessage[ChunkTypeMessageDelta]{
			Type: ChunkTypeMessageDelta("message_delta"),
			Message: &ChunkMessageData{
				StopReason:   stopReason,
				StopSequence: nil,
			},
			Usage: &ChunkMessageUsage{
				InputTokens:          usage.InputTokens,
				CacheReadInputTokens: usage.InputTokensDetails.CachedTokens,
				OutputTokens:         usage.OutputTokens,
			},
		},
	}
}

func (c *NativeResponseChunkToResponseChunkConverter) buildMessageStop() ResponseChunk {
	return ResponseChunk{
		OfMessageStop: &ChunkMessage[ChunkTypeMessageStop]{
			Type: ChunkTypeMessageStop("message_stop"),
		},
	}
}
