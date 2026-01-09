package anthropic_responses

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/llm/constants"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
)

func (in *Request) ToNativeRequest() *responses.Request {
	instructions := ""
	for _, sys := range in.System {
		instructions += sys.Text
	}

	out := &responses.Request{
		Model:        in.Model,
		Input:        MessagesToNativeMessages(in.Messages),
		Tools:        ToolsToNativeTools(in.Tools),
		Instructions: utils.Ptr(instructions),
		Parameters: responses.Parameters{
			Background:      utils.Ptr(false),
			MaxOutputTokens: &in.MaxTokens,
			Temperature:     in.Temperature,
			TopP:            in.TopP,
			TopLogprobs:     in.TopK,
			Metadata:        in.Metadata,
			Stream:          in.Stream,
			Include:         []responses.Includable{},
		},
	}

	if in.Thinking != nil {
		if in.Thinking.Type != nil && *in.Thinking.Type == "enabled" {
			out.Reasoning = &responses.ReasoningParam{
				Summary:      utils.Ptr("auto"),
				BudgetTokens: in.Thinking.BudgetTokens,
			}
			out.Include = append(out.Include, responses.IncludableReasoningEncryptedContent)
		}
	}

	if in.OutputFormat != nil {
		out.Text = &responses.TextFormat{
			Format: in.OutputFormat,
		}
		out.Text.Format["name"] = "structured_output"
	}

	return out
}

func (in Role) ToNativeRole() constants.Role {
	switch in {
	case RoleUser:
		return constants.RoleUser
	case RoleAssistant:
		return constants.RoleAssistant
	}

	return constants.RoleAssistant
}

func ToolsToNativeTools(in []ToolUnion) []responses.ToolUnion {
	out := make([]responses.ToolUnion, len(in))
	for idx, tool := range in {
		out[idx] = tool.ToNative()
	}

	return out
}

func (in *ToolUnion) ToNative() responses.ToolUnion {
	out := responses.ToolUnion{}

	if in.OfCustomTool != nil {
		out.OfFunction = &responses.FunctionTool{
			Type:        "function",
			Name:        in.OfCustomTool.Name,
			Description: in.OfCustomTool.Description,
			Parameters:  in.OfCustomTool.InputSchema,
		}
	}

	if in.OfWebSearchTool != nil {
		out.OfWebSearch = &responses.WebSearchTool{
			Type: "web_search",
			Filters: &responses.WebSearchToolFilters{
				AllowedDomains: in.OfWebSearchTool.AllowedDomains,
			},
		}

		if in.OfWebSearchTool.UserLocation != nil {
			out.OfWebSearch.UserLocation = &responses.WebSearchToolUserLocation{
				Type:     in.OfWebSearchTool.UserLocation.Type,
				Country:  in.OfWebSearchTool.UserLocation.Country,
				City:     in.OfWebSearchTool.UserLocation.City,
				Region:   in.OfWebSearchTool.UserLocation.Region,
				Timezone: in.OfWebSearchTool.UserLocation.Timezone,
			}
		}
	}

	return out
}

func CitationsToNativeAnnotations(citations []Citation) []responses.Annotation {
	var annotations []responses.Annotation

	for _, citation := range citations {
		annotations = append(annotations, responses.Annotation{
			Type:       "url_citation",
			Title:      citation.Title,
			URL:        citation.Url,
			StartIndex: 0,
			EndIndex:   0,
			ExtraParams: map[string]any{
				"Anthropic": citation,
			},
		})
	}

	return annotations
}

func MessagesToNativeMessages(msgs []MessageUnion) responses.InputUnion {
	out := responses.InputUnion{
		OfString:           nil,
		OfInputMessageList: responses.InputMessageList{},
	}

	for _, msg := range msgs {
		out.OfInputMessageList = append(out.OfInputMessageList, msg.ToNativeMessage()...)
	}

	return out
}

func (msg *MessageUnion) ToNativeMessage() []responses.InputMessageUnion {
	out := []responses.InputMessageUnion{}

	var previousServerToolUse *ServerToolUseContent

	for _, content := range msg.Content {
		if content.OfText != nil {
			if content.OfText.Citations != nil {
				out = append(out, responses.InputMessageUnion{
					OfInputMessage: &responses.InputMessage{
						Role: msg.Role.ToNativeRole(),
						Content: responses.InputContent{
							{
								OfOutputText: &responses.OutputTextContent{
									Text:        content.OfText.Text,
									Annotations: CitationsToNativeAnnotations(content.OfText.Citations), // Convert citation to annotation
								},
							},
						},
					},
				})
			} else {
				out = append(out, responses.InputMessageUnion{
					OfInputMessage: &responses.InputMessage{
						Role: msg.Role.ToNativeRole(),
						Content: responses.InputContent{
							{
								OfInputText: &responses.InputTextContent{
									Text: content.OfText.Text,
								},
							},
						},
					},
				})
			}
		}

		if content.OfToolUse != nil {
			argsBuf, err := sonic.Marshal(content.OfToolUse.Input)
			if err != nil {
				argsBuf = []byte("{}")
			}

			out = append(out, responses.InputMessageUnion{
				OfFunctionCall: &responses.FunctionCallMessage{
					ID:        content.OfToolUse.ID,
					CallID:    content.OfToolUse.ID,
					Name:      content.OfToolUse.Name,
					Arguments: string(argsBuf),
				},
			})
		}

		if content.OfToolResult != nil {
			outputs := responses.FunctionCallOutputContentUnion{
				OfString: nil,
				OfList:   responses.InputContent{},
			}

			// TODO: outputContent can be text, image, search result or document
			for _, outputContent := range content.OfToolResult.Content {
				if outputContent.OfText != nil {
					outputs.OfList = append(outputs.OfList, responses.InputContentUnion{
						OfInputText: &responses.InputTextContent{
							Text: outputContent.OfText.Text,
						},
					})
				}
			}

			out = append(out, responses.InputMessageUnion{
				OfFunctionCallOutput: &responses.FunctionCallOutputMessage{
					ID:     content.OfToolResult.ToolUseID,
					CallID: content.OfToolResult.ToolUseID,
					Output: outputs,
				},
			})
		}

		if content.OfThinking != nil {
			out = append(out, responses.InputMessageUnion{
				OfReasoning: &responses.ReasoningMessage{
					ID: uuid.NewString(),
					Summary: []responses.SummaryTextContent{{
						Text: content.OfThinking.Thinking,
					}},
					EncryptedContent: utils.Ptr(content.OfThinking.Signature),
				},
			})
		}

		if content.OfRedactedThinking != nil {
			out = append(out, responses.InputMessageUnion{
				OfReasoning: &responses.ReasoningMessage{
					ID:               uuid.NewString(),
					Summary:          nil,
					EncryptedContent: utils.Ptr(content.OfRedactedThinking.Data),
				},
			})
		}

		if content.OfServerToolUse != nil {
			previousServerToolUse = content.OfServerToolUse
		}

		if content.OfWebSearchResult != nil {
			if previousServerToolUse != nil && previousServerToolUse.Name == "web_search" {
				id := previousServerToolUse.Id
				query := previousServerToolUse.Input.Query
				var sources []responses.WebSearchCallActionOfSearchSource
				for _, searchResultContent := range content.OfWebSearchResult.Content {
					sources = append(sources, responses.WebSearchCallActionOfSearchSource{
						Type: "url",
						URL:  searchResultContent.Url,
						ExtraParams: map[string]any{
							"Anthropic": searchResultContent,
						},
					})
				}

				out = append(out, responses.InputMessageUnion{
					OfWebSearchCall: &responses.WebSearchCallMessage{
						ID: id,
						Action: responses.WebSearchCallActionUnion{
							OfSearch: &responses.WebSearchCallActionOfSearch{
								Queries: []string{
									query,
								},
								Query:   query,
								Sources: sources,
							},
						},
						Status: "completed",
					},
				})

				previousServerToolUse = nil
			}
		}
	}

	return out
}

func (in *Response) ToNativeResponse() *responses.Response {
	output := []responses.OutputMessageUnion{}

	var previousWebSearchCall *ServerToolUseContent

	for _, content := range in.Content {
		if content.OfText != nil {
			output = append(output, responses.OutputMessageUnion{
				OfOutputMessage: &responses.OutputMessage{
					Role: constants.RoleAssistant,
					Content: responses.OutputContent{
						{
							OfOutputText: &responses.OutputTextContent{
								Text:        content.OfText.Text,
								Annotations: CitationsToNativeAnnotations(content.OfText.Citations),
							},
						},
					},
				},
			})
		}

		if content.OfToolUse != nil {
			args, err := sonic.Marshal(content.OfToolUse.Input)
			if err != nil {
				args = []byte("{}")
			}

			output = append(output, responses.OutputMessageUnion{
				OfFunctionCall: &responses.FunctionCallMessage{
					ID:        content.OfToolUse.ID,
					CallID:    content.OfToolUse.ID,
					Name:      content.OfToolUse.Name,
					Arguments: string(args),
				},
			})
		}

		if content.OfThinking != nil {
			output = append(output, responses.OutputMessageUnion{
				OfReasoning: &responses.ReasoningMessage{
					Summary: []responses.SummaryTextContent{
						{Text: content.OfThinking.Thinking},
					},
					EncryptedContent: utils.Ptr(content.OfThinking.Signature),
				},
			})
		}

		// Redacted thinking is also converted into reasoning message only.
		// Just that it won't have a summary.
		if content.OfRedactedThinking != nil {
			output = append(output, responses.OutputMessageUnion{
				OfReasoning: &responses.ReasoningMessage{
					EncryptedContent: utils.Ptr(content.OfThinking.Signature),
				},
			})
		}

		// We simply store the service_tool_use message for later reference
		if content.OfServerToolUse != nil {
			previousWebSearchCall = content.OfServerToolUse
		}

		if content.OfWebSearchResult != nil && previousWebSearchCall != nil {
			sources := []responses.WebSearchCallActionOfSearchSource{}
			for _, searchResultContent := range content.OfWebSearchResult.Content {
				sources = append(sources, responses.WebSearchCallActionOfSearchSource{
					Type: "url",
					URL:  searchResultContent.Url,
					ExtraParams: map[string]any{
						"Anthropic": searchResultContent,
					},
				})
			}

			output = append(output, responses.OutputMessageUnion{
				OfWebSearchCall: &responses.WebSearchCallMessage{
					ID: previousWebSearchCall.Id,
					Action: responses.WebSearchCallActionUnion{
						OfSearch: &responses.WebSearchCallActionOfSearch{
							Queries: []string{
								previousWebSearchCall.Input.Query,
							},
							Query:   previousWebSearchCall.Input.Query,
							Sources: sources,
						},
					},
					Status: "completed",
				},
			})

			previousWebSearchCall = nil
		}
	}

	return &responses.Response{
		ID:     in.Id,
		Model:  in.Model,
		Output: output,
		Usage: &responses.Usage{
			InputTokens: in.Usage.InputTokens,
			InputTokensDetails: struct {
				CachedTokens int `json:"cached_tokens"`
			}{
				CachedTokens: in.Usage.CacheReadInputTokens,
			},
			OutputTokens: in.Usage.OutputTokens,
			OutputTokensDetails: struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			}{},
			TotalTokens: in.Usage.InputTokens + in.Usage.OutputTokens,
		},
		Error: &responses.Error{
			Type:    "",
			Message: "",
			Param:   "",
			Code:    "",
		},
		ServiceTier: in.ServiceTier,
		Metadata: map[string]any{
			"stop_reason":   in.StopReason,
			"stop_sequence": in.StopSequence,
		},
	}
}

// =============================================================================
// ResponseChunk to Native Conversion
// =============================================================================

// ResponseChunkToNativeResponseChunkConverter converts Anthropic stream chunks to native format.
// It maintains state across chunk conversions to accumulate deltas and track the current content block.
type ResponseChunkToNativeResponseChunkConverter struct {
	// State from message_start - contains message ID, model, role
	messageStart *ChunkMessage[ChunkTypeMessageStart]
	// State from message_delta - contains usage info
	messageDelta *ChunkMessage[ChunkTypeMessageDelta]
	// Current content block being processed
	currentBlock *ChunkContentBlock[ChunkTypeContentBlockStart]

	// Tracking
	sequenceNumber   int
	outputIndex      int
	contentIndex     int // Always 0, as each Anthropic content becomes a separate native message
	currentOutputID  string
	accumulatedDelta string
	accumulatedSig   string // Accumulated reasoning signature
	completedOutputs []responses.OutputMessageUnion
}

// nextSeqNum returns the next sequence number and increments the counter.
func (c *ResponseChunkToNativeResponseChunkConverter) nextSeqNum() int {
	n := c.sequenceNumber
	c.sequenceNumber++
	return n
}

// currentRole returns the role from the message start, defaulting to assistant.
func (c *ResponseChunkToNativeResponseChunkConverter) currentRole() constants.Role {
	if c.messageStart != nil && c.messageStart.Message != nil {
		return c.messageStart.Message.Role.ToNativeRole()
	}
	return constants.RoleAssistant
}

// ResponseChunkToNativeResponseChunk converts a single Anthropic chunk to zero or more native chunks.
func (c *ResponseChunkToNativeResponseChunkConverter) ResponseChunkToNativeResponseChunk(in *ResponseChunk) []*responses.ResponseChunk {
	if in == nil {
		return nil
	}

	switch {
	case in.OfMessageStart != nil:
		return c.handleMessageStart(in.OfMessageStart)
	case in.OfContentBlockStart != nil:
		return c.handleContentBlockStart(in.OfContentBlockStart)
	case in.OfContentBlockDelta != nil:
		return c.handleContentBlockDelta(in.OfContentBlockDelta)
	case in.OfContentBlockStop != nil:
		return c.handleContentBlockStop()
	case in.OfMessageDelta != nil:
		return c.handleMessageDelta(in.OfMessageDelta)
	case in.OfMessageStop != nil:
		return c.handleMessageStop()
	case in.OfPing != nil:
		return nil // Ping is keep-alive, no conversion needed
	}

	return nil
}

// =============================================================================
// Event Handlers
// =============================================================================

// handleMessageStart emits response.created and response.in_progress
func (c *ResponseChunkToNativeResponseChunkConverter) handleMessageStart(msg *ChunkMessage[ChunkTypeMessageStart]) []*responses.ResponseChunk {
	c.messageStart = msg
	msgData := msg.Message

	return []*responses.ResponseChunk{
		c.buildResponseCreated(msgData.Id, msgData.Model),
		c.buildResponseInProgress(msgData.Id),
	}
}

// handleContentBlockStart emits output_item.added (and content_part.added for text/reasoning)
func (c *ResponseChunkToNativeResponseChunkConverter) handleContentBlockStart(block *ChunkContentBlock[ChunkTypeContentBlockStart]) []*responses.ResponseChunk {
	c.currentBlock = block
	content := block.ContentBlock

	switch {
	case content.OfText != nil:
		c.currentOutputID = responses.NewOutputItemMessageID()
		return c.handleTextBlockStart()
	case content.OfToolUse != nil:
		c.currentOutputID = responses.NewOutputItemFunctionCallID()
		return c.handleToolUseBlockStart(content.OfToolUse)
	case content.OfThinking != nil:
		c.currentOutputID = responses.NewOutputItemReasoningID()
		return c.handleThinkingBlockStart(content.OfThinking)
	case content.OfServerToolUse != nil:
		c.currentOutputID = content.OfServerToolUse.Id
		return c.handleServerToolUseBlockStart(content.OfServerToolUse)
	case content.OfWebSearchResult != nil:
		return c.handleWebSearchToolResultBlockStart(content.OfWebSearchResult)
	}

	return nil
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleTextBlockStart() []*responses.ResponseChunk {
	return []*responses.ResponseChunk{
		c.buildOutputItemAddedMessage(),
		c.buildContentPartAddedText(),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleToolUseBlockStart(toolUse *ToolUseContent) []*responses.ResponseChunk {
	args, _ := sonic.Marshal(toolUse.Input)
	if args == nil {
		args = []byte("{}")
	}
	return []*responses.ResponseChunk{
		c.buildOutputItemAddedFunctionCall(toolUse.ID, toolUse.Name, string(args)),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleThinkingBlockStart(thinking *ThinkingContent) []*responses.ResponseChunk {
	return []*responses.ResponseChunk{
		c.buildOutputItemAddedReasoning(thinking.Signature),
		c.buildReasoningSummaryPartAdded(),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleServerToolUseBlockStart(serverToolUse *ServerToolUseContent) []*responses.ResponseChunk {
	return []*responses.ResponseChunk{
		c.buildOutputItemAddedWebSearchCall(),
		c.buildWebSearchCallInProgress(),
		c.buildWebSearchCallSearching(),
		c.buildWebSearchCallCompleted(),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleWebSearchToolResultBlockStart(webResultToolResult *WebSearchResultContent) []*responses.ResponseChunk {
	return nil
}

// handleContentBlockDelta emits delta chunks based on the current block type
func (c *ResponseChunkToNativeResponseChunkConverter) handleContentBlockDelta(delta *ChunkContentBlock[ChunkTypeContentBlockDelta]) []*responses.ResponseChunk {
	if c.currentBlock == nil || c.currentBlock.ContentBlock == nil {
		return nil
	}

	content := c.currentBlock.ContentBlock

	switch {
	case content.OfText != nil && delta.Delta.OfText != nil:
		text := delta.Delta.OfText.Text
		c.accumulatedDelta += text
		return []*responses.ResponseChunk{c.buildOutputTextDelta(text)}

	case content.OfText != nil && delta.Delta.OfCitation != nil:
		citation := delta.Delta.OfCitation.Citation
		return []*responses.ResponseChunk{
			c.buildOutputTextAnnotationAdded(citation),
		}

	case content.OfToolUse != nil && delta.Delta.OfInputJSON != nil:
		json := delta.Delta.OfInputJSON.PartialJSON
		c.accumulatedDelta += json
		return []*responses.ResponseChunk{c.buildFunctionCallArgumentsDelta(json)}

	case content.OfServerToolUse != nil && delta.Delta.OfInputJSON != nil:
		json := delta.Delta.OfInputJSON.PartialJSON
		c.accumulatedDelta += json
		return nil

	case content.OfThinking != nil:
		return c.handleThinkingDelta(delta.Delta)
	}

	return nil
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleThinkingDelta(delta *ChunkContentBlockDeltaUnion) []*responses.ResponseChunk {
	if delta.OfThinking != nil {
		text := delta.OfThinking.Thinking
		c.accumulatedDelta += text
		return []*responses.ResponseChunk{c.buildReasoningSummaryTextDelta(text)}
	}
	if delta.OfThinkingSignature != nil {
		sig := delta.OfThinkingSignature.Signature
		c.accumulatedSig += sig
		return []*responses.ResponseChunk{c.buildReasoningSignatureDelta(sig)}
	}
	return nil
}

// handleContentBlockStop emits done chunks and stores the completed output
func (c *ResponseChunkToNativeResponseChunkConverter) handleContentBlockStop() []*responses.ResponseChunk {
	if c.currentBlock == nil || c.currentBlock.ContentBlock == nil {
		return nil
	}

	var result []*responses.ResponseChunk
	content := c.currentBlock.ContentBlock

	switch {
	case content.OfText != nil:
		result = c.completeTextBlock()
	case content.OfToolUse != nil:
		result = c.completeToolUseBlock(content.OfToolUse)
	case content.OfThinking != nil:
		result = c.completeThinkingBlock()
	case content.OfServerToolUse != nil:
		return nil // we don't do anything for server_tool_use content stop, we will wait for content_block_stop of "web_search_tool_result"
	case content.OfWebSearchResult != nil:
		result = c.completeWebSearchCallBlock(content.OfWebSearchResult)
	}

	// Reset for next block
	c.outputIndex++
	c.accumulatedDelta = ""
	c.accumulatedSig = ""

	return result
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeTextBlock() []*responses.ResponseChunk {
	text := c.accumulatedDelta
	role := c.currentRole()

	// Store for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfOutputMessage: &responses.OutputMessage{
			ID:   c.currentOutputID,
			Role: role,
			Content: responses.OutputContent{
				{OfOutputText: &responses.OutputTextContent{Text: text}},
			},
		},
	})

	return []*responses.ResponseChunk{
		c.buildOutputTextDone(text),
		c.buildContentPartDoneText(text),
		c.buildOutputItemDoneMessage(text, role),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeToolUseBlock(toolUse *ToolUseContent) []*responses.ResponseChunk {
	args := c.accumulatedDelta
	if args == "" {
		args = "{}"
	}

	// Store for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfFunctionCall: &responses.FunctionCallMessage{
			ID:        c.currentOutputID,
			CallID:    toolUse.ID,
			Name:      toolUse.Name,
			Arguments: args,
		},
	})

	return []*responses.ResponseChunk{
		c.buildFunctionCallArgumentsDone(args),
		c.buildOutputItemDoneFunctionCall(toolUse.ID, toolUse.Name, args),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeThinkingBlock() []*responses.ResponseChunk {
	text := c.accumulatedDelta
	sig := c.accumulatedSig

	// Store for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfReasoning: &responses.ReasoningMessage{
			ID:               c.currentOutputID,
			Summary:          []responses.SummaryTextContent{{Text: text}},
			EncryptedContent: utils.Ptr(sig),
		},
	})

	return []*responses.ResponseChunk{
		c.buildReasoningSummaryTextDone(text),
		c.buildReasoningSummaryPartDone(text),
		c.buildOutputItemDoneReasoning(text, sig),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeWebSearchCallBlock(webSearchResult *WebSearchResultContent) []*responses.ResponseChunk {
	return []*responses.ResponseChunk{
		c.buildOutputItemDoneWebSearchCall(webSearchResult),
	}
}

// handleMessageDelta stores usage info for the final response
func (c *ResponseChunkToNativeResponseChunkConverter) handleMessageDelta(delta *ChunkMessage[ChunkTypeMessageDelta]) []*responses.ResponseChunk {
	c.messageDelta = delta
	return nil
}

// handleMessageStop emits response.completed
func (c *ResponseChunkToNativeResponseChunkConverter) handleMessageStop() []*responses.ResponseChunk {
	return []*responses.ResponseChunk{c.buildResponseCompleted()}
}

// =============================================================================
// Chunk Builders
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) buildResponseCreated(id, model string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfResponseCreated: &responses.ChunkResponse[constants.ChunkTypeResponseCreated]{
			Type:           constants.ChunkTypeResponseCreated(""),
			SequenceNumber: c.nextSeqNum(),
			Response: responses.ChunkResponseData{
				Id:         id,
				Object:     "response",
				CreatedAt:  int(time.Now().Unix()),
				Status:     "in_progress",
				Background: false,
				Request:    responses.Request{Model: model},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildResponseInProgress(id string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfResponseInProgress: &responses.ChunkResponse[constants.ChunkTypeResponseInProgress]{
			Type:           constants.ChunkTypeResponseInProgress(""),
			SequenceNumber: c.nextSeqNum(),
			Response: responses.ChunkResponseData{
				Id:         id,
				Object:     "response",
				CreatedAt:  int(time.Now().Unix()),
				Status:     "in_progress",
				Background: false,
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedMessage() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:    "message",
				Id:      c.currentOutputID,
				Status:  "in_progress",
				Role:    c.currentRole(),
				Content: responses.OutputContent{},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedFunctionCall(callID, name, args string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:      "function_call",
				Id:        c.currentOutputID,
				Status:    "in_progress",
				CallID:    utils.Ptr(callID),
				Name:      utils.Ptr(name),
				Arguments: utils.Ptr(args),
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedReasoning(signature string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:             "reasoning",
				Id:               c.currentOutputID,
				Status:           "in_progress",
				Summary:          []responses.SummaryTextContent{},
				EncryptedContent: utils.Ptr(signature),
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedWebSearchCall() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:   "web_search_call",
				Id:     c.currentOutputID,
				Status: "in_progress",
				Action: &responses.WebSearchCallActionUnion{
					OfSearch: &responses.WebSearchCallActionOfSearch{},
				},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildWebSearchCallInProgress() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfWebSearchCallInProgress: &responses.ChunkWebSearchCall[constants.ChunkTypeWebSearchCallInProgress]{
			Type:           constants.ChunkTypeWebSearchCallInProgress(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildWebSearchCallSearching() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfWebSearchCallSearching: &responses.ChunkWebSearchCall[constants.ChunkTypeWebSearchCallSearching]{
			Type:           constants.ChunkTypeWebSearchCallSearching(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildWebSearchCallCompleted() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfWebSearchCallCompleted: &responses.ChunkWebSearchCall[constants.ChunkTypeWebSearchCallCompleted]{
			Type:           constants.ChunkTypeWebSearchCallCompleted(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildContentPartAddedText() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfContentPartAdded: &responses.ChunkContentPart[constants.ChunkTypeContentPartAdded]{
			Type:           constants.ChunkTypeContentPartAdded(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Part:           responses.OutputContentUnion{OfOutputText: &responses.OutputTextContent{Text: ""}},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryPartAdded() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryPartAdded: &responses.ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartAdded]{
			Type:           constants.ChunkTypeReasoningSummaryPartAdded(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Part:           responses.SummaryTextContent{Text: ""},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputTextDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputTextDelta: &responses.ChunkOutputText[constants.ChunkTypeOutputTextDelta]{
			Type:           constants.ChunkTypeOutputTextDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Delta:          delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputTextAnnotationAdded(citation Citation) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputTextAnnotationAdded: &responses.ChunkOutputText[constants.ChunkTypeOutputTextAnnotationAdded]{
			Type:           constants.ChunkTypeOutputTextAnnotationAdded(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Annotation: responses.Annotation{
				Type:       "url_citation",
				Title:      citation.Title,
				URL:        citation.Url,
				StartIndex: 0,
				EndIndex:   0,
				ExtraParams: map[string]any{
					"Anthropic": citation,
				},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildFunctionCallArgumentsDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfFunctionCallArgumentsDelta: &responses.ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDelta]{
			Type:           constants.ChunkTypeFunctionCallArgumentsDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			Delta:          delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryTextDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryTextDelta: &responses.ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDelta]{
			Type:           constants.ChunkTypeReasoningSummaryTextDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Delta:          delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSignatureDelta(sig string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryTextDelta: &responses.ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDelta]{
			Type:             constants.ChunkTypeReasoningSummaryTextDelta(""),
			SequenceNumber:   c.nextSeqNum(),
			ItemId:           c.currentOutputID,
			OutputIndex:      c.outputIndex,
			SummaryIndex:     c.contentIndex,
			EncryptedContent: utils.Ptr(sig),
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputTextDone(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputTextDone: &responses.ChunkOutputText[constants.ChunkTypeOutputTextDone]{
			Type:           constants.ChunkTypeOutputTextDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Text:           utils.Ptr(text),
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildContentPartDoneText(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfContentPartDone: &responses.ChunkContentPart[constants.ChunkTypeContentPartDone]{
			Type:           constants.ChunkTypeContentPartDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Part:           responses.OutputContentUnion{OfOutputText: &responses.OutputTextContent{Text: text}},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneMessage(text string, role constants.Role) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:    "message",
				Id:      c.currentOutputID,
				Status:  "completed",
				Role:    role,
				Content: responses.OutputContent{{OfOutputText: &responses.OutputTextContent{Text: text}}},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildFunctionCallArgumentsDone(args string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfFunctionCallArgumentsDone: &responses.ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDone]{
			Type:           constants.ChunkTypeFunctionCallArgumentsDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			Arguments:      args,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneFunctionCall(callID, name, args string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:      "function_call",
				Id:        c.currentOutputID,
				Status:    "completed",
				CallID:    utils.Ptr(callID),
				Name:      utils.Ptr(name),
				Arguments: utils.Ptr(args),
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryTextDone(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryTextDone: &responses.ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDone]{
			Type:           constants.ChunkTypeReasoningSummaryTextDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Text:           utils.Ptr(text),
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryPartDone(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryPartDone: &responses.ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartDone]{
			Type:           constants.ChunkTypeReasoningSummaryPartDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.currentOutputID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Part:           responses.SummaryTextContent{Text: text},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneReasoning(text, sig string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:             "reasoning",
				Id:               c.currentOutputID,
				Status:           "completed",
				EncryptedContent: utils.Ptr(sig),
				Summary:          []responses.SummaryTextContent{{Text: text}},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneWebSearchCall(webSearchResults *WebSearchResultContent) *responses.ResponseChunk {
	sources := []responses.WebSearchCallActionOfSearchSource{}
	for _, webSearchResult := range webSearchResults.Content {
		sources = append(sources, responses.WebSearchCallActionOfSearchSource{
			Type: "url",
			URL:  webSearchResult.Url,
			ExtraParams: map[string]any{
				"Anthropic": webSearchResult,
			},
		})
	}

	query := ""
	accumulatedPayload := struct {
		Query string `json:"query"`
	}{}
	if err := sonic.Unmarshal([]byte(c.accumulatedDelta), &accumulatedPayload); err == nil {
		query = accumulatedPayload.Query
	}

	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:   "web_search_call",
				Id:     c.currentOutputID,
				Status: "completed",
				Action: &responses.WebSearchCallActionUnion{
					OfSearch: &responses.WebSearchCallActionOfSearch{
						Queries: []string{query},
						Query:   query,
						Sources: sources,
					},
				},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildResponseCompleted() *responses.ResponseChunk {
	msg := c.messageStart.Message
	usage := c.messageDelta.Usage

	return &responses.ResponseChunk{
		OfResponseCompleted: &responses.ChunkResponse[constants.ChunkTypeResponseCompleted]{
			Type:           constants.ChunkTypeResponseCompleted(""),
			SequenceNumber: c.nextSeqNum(),
			Response: responses.ChunkResponseData{
				Id:        msg.Id,
				Object:    "response",
				CreatedAt: int(time.Now().Unix()),
				Status:    "completed",
				Output:    c.completedOutputs,
				Usage: responses.Usage{
					InputTokens: usage.InputTokens,
					InputTokensDetails: struct {
						CachedTokens int `json:"cached_tokens"`
					}{CachedTokens: usage.CacheReadInputTokens},
					OutputTokens: usage.OutputTokens,
					TotalTokens:  usage.InputTokens + usage.OutputTokens,
					OutputTokensDetails: struct {
						ReasoningTokens int `json:"reasoning_tokens"`
					}{ReasoningTokens: 0},
				},
				Request: responses.Request{Model: msg.Model},
			},
		},
	}
}
