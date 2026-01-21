package gemini_responses

import (
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/llm/constants"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/google/uuid"
)

func (in *Request) ToNativeRequest() *responses.Request {
	out := &responses.Request{
		Model:        in.Model,
		Input:        MessagesToNativeMessages(in.Contents),
		Instructions: utils.Ptr(in.SystemInstruction.String()),
		Tools:        ToolsToNativeTools(in.Tools),
		Parameters: responses.Parameters{
			Background:        nil,
			MaxOutputTokens:   in.GenerationConfig.MaxOutputTokens,
			MaxToolCalls:      nil,
			ParallelToolCalls: nil,
			Store:             nil,
			Temperature:       in.GenerationConfig.Temperature,
			TopLogprobs:       in.GenerationConfig.TopK,
			TopP:              in.GenerationConfig.TopP,
			Include:           nil,
			Metadata:          nil,
			Stream:            in.Stream,
		},
	}

	includables := []responses.Includable{}
	if in.GenerationConfig.ThinkingConfig != nil {
		effort := "high"
		if in.GenerationConfig.ThinkingConfig.ThinkingLevel != nil && *in.GenerationConfig.ThinkingConfig.ThinkingLevel != "HIGH" {
			effort = "low"
		}

		out.Reasoning = &responses.ReasoningParam{
			Effort:       &effort,
			Summary:      utils.Ptr("auto"),
			BudgetTokens: in.GenerationConfig.ThinkingConfig.ThinkingBudget,
		}

		includables = append(includables, responses.IncludableReasoningEncryptedContent)
	}

	if len(includables) > 0 {
		out.Include = includables
	}

	if (out.Tools == nil || len(out.Tools) == 0) && in.GenerationConfig.ResponseJsonSchema != nil {
		out.Text = &responses.TextFormat{
			Format: map[string]any{
				"type":   "json_schema",
				"name":   "structured_output",
				"strict": false,
				"schema": in.GenerationConfig.ResponseJsonSchema,
			},
		}
	}

	return out
}

func (in Role) ToNativeRole() constants.Role {
	switch in {
	case RoleUser:
		return constants.RoleUser
	case RoleModel:
		return constants.RoleAssistant
	case RoleSystem:
		return constants.RoleSystem
	}

	return constants.RoleAssistant
}

func ToolsToNativeTools(in []Tool) []responses.ToolUnion {
	out := []responses.ToolUnion{}

	for _, tool := range in {
		out = append(out, tool.ToNative()...)
	}

	return out
}

func (in *Tool) ToNative() []responses.ToolUnion {
	out := []responses.ToolUnion{}

	if in.FunctionDeclarations != nil {
		for _, fnDecl := range in.FunctionDeclarations {
			out = append(out, responses.ToolUnion{
				OfFunction: &responses.FunctionTool{
					Type:        "function",
					Name:        fnDecl.Name,
					Description: utils.Ptr(fnDecl.Description),
					Parameters:  fnDecl.ParametersJsonSchema,
				},
			})
		}
	}

	if in.CodeExecution != nil {
		out = append(out, responses.ToolUnion{
			OfCodeExecution: &responses.CodeExecutionTool{},
		})
	}

	return out
}

func MessagesToNativeMessages(msgs []Content) responses.InputUnion {
	out := responses.InputUnion{
		OfString:           nil,
		OfInputMessageList: responses.InputMessageList{},
	}

	for _, content := range msgs {
		out.OfInputMessageList = append(out.OfInputMessageList, content.ToNativeMessage()...)
	}

	return out
}

func (content *Content) ToNativeMessage() []responses.InputMessageUnion {
	out := []responses.InputMessageUnion{}

	var previousExecutableCodePart *ExecutableCodePart
	for _, part := range content.Parts {
		if part.Text != nil {
			out = append(out, responses.InputMessageUnion{
				OfInputMessage: &responses.InputMessage{
					Role: content.Role.ToNativeRole(),
					Content: responses.InputContent{
						{
							OfInputText: &responses.InputTextContent{
								Type: "input_text",
								Text: *part.Text,
							},
						},
					},
				},
			})
		}

		if part.FunctionCall != nil {
			args, err := sonic.Marshal(part.FunctionCall.Args)
			if err != nil {
				args = []byte("{}")
			}

			out = append(out, responses.InputMessageUnion{
				OfFunctionCall: &responses.FunctionCallMessage{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}

		if part.FunctionResponse != nil {
			for _, v := range part.FunctionResponse.Response {
				out = append(out, responses.InputMessageUnion{
					OfFunctionCallOutput: &responses.FunctionCallOutputMessage{
						ID:     part.FunctionResponse.ID,
						CallID: part.FunctionResponse.ID,
						Output: responses.FunctionCallOutputContentUnion{
							OfString: utils.Ptr(v.(string)),
							OfList:   responses.InputContent{},
						},
					},
				})
			}
		}

		if part.ExecutableCode != nil {
			previousExecutableCodePart = part.ExecutableCode
		}

		if previousExecutableCodePart != nil && part.CodeExecutionResult != nil {
			out = append(out, responses.InputMessageUnion{
				OfCodeInterpreterCall: &responses.CodeInterpreterCallMessage{
					Status: "completed",
					Code:   previousExecutableCodePart.Code,
					Outputs: []responses.CodeInterpreterCallOutputParam{
						{
							Type: "logs",
							Logs: part.CodeExecutionResult.Output,
						},
					},
				},
			})
			previousExecutableCodePart = nil
		}
	}

	return out
}

func (in *Response) ToNativeResponse() *responses.Response {
	output := []responses.OutputMessageUnion{}

	var previousExecutableCodePart *ExecutableCodePart
	for _, part := range in.Candidates[0].Content.Parts {
		if part.Text != nil {
			output = append(output, responses.OutputMessageUnion{
				OfOutputMessage: &responses.OutputMessage{
					Role: constants.RoleAssistant,
					Content: responses.OutputContent{
						{
							OfOutputText: &responses.OutputTextContent{
								Text: *part.Text,
							},
						},
					},
				},
			})
		}

		if part.FunctionCall != nil {
			args, err := sonic.Marshal(part.FunctionCall.Args)
			if err != nil {
				args = []byte("{}")
			}

			callId := uuid.NewString()
			output = append(output, responses.OutputMessageUnion{
				OfFunctionCall: &responses.FunctionCallMessage{
					ID:        callId,
					CallID:    callId,
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}

		if part.ExecutableCode != nil {
			previousExecutableCodePart = part.ExecutableCode
		}

		if part.CodeExecutionResult != nil && previousExecutableCodePart != nil {
			output = append(output, responses.OutputMessageUnion{
				OfCodeInterpreterCall: &responses.CodeInterpreterCallMessage{
					Code: previousExecutableCodePart.Code,
					Outputs: []responses.CodeInterpreterCallOutputParam{
						{
							Type: "logs",
							Logs: part.CodeExecutionResult.Output,
						},
					},
				},
			})
			previousExecutableCodePart = nil
		}
	}

	return &responses.Response{
		ID:     in.ResponseID,
		Model:  in.ModelVersion,
		Output: output,
		Usage: &responses.Usage{
			InputTokens: in.UsageMetadata.PromptTokenCount,
			InputTokensDetails: struct {
				CachedTokens int `json:"cached_tokens"`
			}{},
			OutputTokens: in.UsageMetadata.CandidatesTokenCount,
			OutputTokensDetails: struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			}{
				ReasoningTokens: in.UsageMetadata.ThoughtsTokenCount,
			},
			TotalTokens: in.UsageMetadata.TotalTokenCount,
		},
		Error:       nil,
		ServiceTier: "",
		Metadata: map[string]any{
			"stop_reason": in.Candidates[0].FinishReason,
		},
	}
}

// =============================================================================
// Gemini ResponseChunk to Native Conversion
// =============================================================================

// ResponseChunkToNativeResponseChunkConverter converts Gemini stream chunks to native format.
// Gemini streams parts within Response objects, unlike Anthropic's event-based streaming.
type ResponseChunkToNativeResponseChunkConverter struct {
	// Stream lifecycle
	streamStarted bool
	streamEnded   bool

	// Current output item state
	currentBlock     *Part
	outputItemActive bool
	outputItemID     string
	outputIndex      int
	contentIndex     int

	// For detecting content type transitions
	previousPart *Part

	// Accumulation
	accumulatedData  string
	completedOutputs []responses.OutputMessageUnion

	// Message-level state
	sequenceNumber int
	messageID      string
	usage          UsageMetadata
	model          string
}

// nextSeqNum returns the next sequence number and increments the counter.
func (c *ResponseChunkToNativeResponseChunkConverter) nextSeqNum() int {
	n := c.sequenceNumber
	c.sequenceNumber++
	return n
}

// getPartType returns the type of a part for transition detection.
func (c *ResponseChunkToNativeResponseChunkConverter) getPartType(part *Part) string {
	switch {
	case part.Text != nil:
		if part.IsThought() {
			return "thought"
		}
		return "text"
	case part.FunctionCall != nil:
		return "function_call"
	case part.InlineData != nil:
		if strings.HasPrefix(part.InlineData.MimeType, "image") {
			return "image_generation_call"
		}
	case part.ExecutableCode != nil && part.CodeExecutionResult != nil:
		return "code_execution"
	}

	return ""
}

// ResponseChunkToNativeResponseChunk converts a Gemini response chunk to native format.
// Pass nil to signal end of stream and emit completion events.
func (c *ResponseChunkToNativeResponseChunkConverter) ResponseChunkToNativeResponseChunk(in *Response) []*responses.ResponseChunk {
	// Stream already ended, ignore further input
	if c.streamEnded {
		return nil
	}

	// nil input signals end of stream
	if in == nil {
		return c.handleStreamEnd()
	}

	// Update usage and model from each chunk (Gemini sends these with every chunk)
	c.usage = *in.UsageMetadata
	c.model = in.ResponseID

	var out []*responses.ResponseChunk

	// Emit stream start events on first chunk
	if !c.streamStarted {
		out = append(out, c.emitStreamStart(in)...)
	}

	// Process all parts in this chunk
	for i := range in.Candidates[0].Content.Parts {
		part := &in.Candidates[0].Content.Parts[i]
		out = append(out, c.handlePart(part)...)
	}

	return out
}

// =============================================================================
// Stream Lifecycle Handlers
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) emitStreamStart(in *Response) []*responses.ResponseChunk {
	c.streamStarted = true
	c.messageID = in.ResponseID

	return []*responses.ResponseChunk{
		c.buildResponseCreated(in.ResponseID, in.ModelVersion),
		c.buildResponseInProgress(in.ResponseID),
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleStreamEnd() []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Complete any active output item
	if c.previousPart != nil {
		out = append(out, c.completeCurrentPart()...)
	}

	// Emit response.completed
	out = append(out, c.buildResponseCompleted())
	c.streamEnded = true

	return out
}

// =============================================================================
// Part Handlers
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) handlePart(part *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Check if it is an empty part
	if part.Text != nil && *part.Text == "" {
		return []*responses.ResponseChunk{}
	}

	// Check if we need to complete previous part (content type changed)
	if c.shouldEndPreviousPart(part) {
		out = append(out, c.completeCurrentPart()...)
		c.outputItemActive = false
		c.accumulatedData = ""
	}

	// Store current block for later reference (used in completion)
	if !c.outputItemActive {
		c.currentBlock = part
		c.outputItemID = uuid.NewString()
	}

	// Handle based on part type
	switch {
	case part.Text != nil:
		if part.IsThought() {
			out = append(out, c.handleThoughtPart(part)...)
		} else {
			out = append(out, c.handleTextPart(part)...)
		}
	case part.FunctionCall != nil:
		out = append(out, c.handleFunctionCallPart(part)...)

	case part.InlineData != nil:
		if strings.HasPrefix(part.InlineData.MimeType, "image") {
			out = append(out, c.handleInlineImageDataPart(part)...)
		}

	case part.ExecutableCode != nil:
		out = append(out, c.handleExecutableCodePart(part)...)

	case part.CodeExecutionResult != nil:
		out = append(out, c.handleCodeExecutionResultPart(part)...)
	}

	c.outputItemActive = true
	c.previousPart = part

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) shouldEndPreviousPart(part *Part) bool {
	if c.previousPart == nil {
		return false
	}
	return c.getPartType(c.previousPart) != c.getPartType(part)
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeCurrentPart() []*responses.ResponseChunk {
	if c.previousPart == nil {
		return nil
	}

	switch {
	case c.previousPart.Text != nil:
		if c.previousPart.IsThought() {
			return c.completeThoughtPart()
		} else {
			return c.completeTextPart()
		}
	case c.previousPart.FunctionCall != nil:
		return c.completeFunctionCallPart()

	case c.previousPart.InlineData != nil:
		if strings.HasPrefix(c.previousPart.InlineData.MimeType, "image") {
			return c.completeInlineImageDataPart()
		}

	case c.previousPart.CodeExecutionResult != nil:
		return c.completeCodeExecutionResult()
	}

	return nil
}

// =============================================================================
// Text Part Handling
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) handleTextPart(part *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Emit start events if this is a new output item
	if !c.outputItemActive {
		out = append(out,
			c.buildOutputItemAddedMessage(),
			c.buildContentPartAddedText(),
		)
	}

	// Avoid emitting empty text if thought signature is present
	if (part.Text == nil || *part.Text == "") && (part.ThoughtSignature != nil && *part.ThoughtSignature != "") {
		return out
	}

	// Emit delta
	out = append(out, c.buildOutputTextDelta(*part.Text))
	c.accumulatedData += *part.Text

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeTextPart() []*responses.ResponseChunk {
	text := c.accumulatedData

	// Store completed output for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfOutputMessage: &responses.OutputMessage{
			ID:   c.outputItemID,
			Role: RoleModel.ToNativeRole(),
			Content: responses.OutputContent{
				{OfOutputText: &responses.OutputTextContent{Text: text}},
			},
		},
	})

	return []*responses.ResponseChunk{
		c.buildOutputTextDone(text),
		c.buildContentPartDoneText(text),
		c.buildOutputItemDoneMessage(text),
	}
}

// =============================================================================
// Function Call Part Handling
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) handleFunctionCallPart(part *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	args, _ := sonic.Marshal(part.FunctionCall.Args)
	if args == nil {
		args = []byte("{}")
	}
	argsStr := string(args)

	// Emit start events if this is a new output item
	if !c.outputItemActive {
		callID := uuid.NewString() + "_" + part.FunctionCall.Name
		out = append(out, c.buildOutputItemAddedFunctionCall(callID, part.FunctionCall.Name, argsStr, part.ThoughtSignature))
	}

	// Emit delta
	out = append(out, c.buildFunctionCallArgumentsDelta(argsStr))
	c.accumulatedData += argsStr

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeFunctionCallPart() []*responses.ResponseChunk {
	args := c.accumulatedData
	if args == "" {
		args = "{}"
	}

	callID := uuid.NewString()
	fnName := c.currentBlock.FunctionCall.Name

	// Store completed output for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfFunctionCall: &responses.FunctionCallMessage{
			ID:        c.outputItemID,
			CallID:    callID,
			Name:      fnName,
			Arguments: args,
		},
	})

	return []*responses.ResponseChunk{
		c.buildFunctionCallArgumentsDone(args),
		c.buildOutputItemDoneFunctionCall(callID, fnName, args),
	}
}

// =============================================================================
// Thought Part Handling
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) handleThoughtPart(part *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Emit start events if this is a new output item
	if !c.outputItemActive {
		out = append(out,
			c.buildOutputItemAddedReasoning(),
			c.buildReasoningSummaryPartAdded(),
		)
	}

	// Emit delta
	out = append(out, c.buildReasoningSummaryTextDelta(*part.Text))
	c.accumulatedData += *part.Text

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeThoughtPart() []*responses.ResponseChunk {
	text := c.accumulatedData

	// Store completed output for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfReasoning: &responses.ReasoningMessage{
			ID: c.outputItemID,
			Summary: []responses.SummaryTextContent{
				{Text: text},
			},
			EncryptedContent: nil,
		},
	})

	return []*responses.ResponseChunk{
		c.buildReasoningSummaryTextDone(text),
		c.buildReasoningSummaryPartDone(text),
		c.buildOutputItemDoneReasoningSummary(text),
	}
}

// =============================================================================
// Inline Data Handling
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) handleInlineImageDataPart(part *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Emit start events if this is a new output item
	if !c.outputItemActive {
		out = append(out,
			c.buildOutputItemAddedImageGenerationCall(),
			c.buildImageGenerationCallInProgress(),
			c.buildImageGenerationCallGenerating(),
		)
	}

	// Emit delta
	out = append(out, c.buildImageGenerationCallPartialImage(part.InlineData.MimeType, part.InlineData.Data))
	c.accumulatedData = part.InlineData.Data

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeInlineImageDataPart() []*responses.ResponseChunk {
	// Store completed output for final response
	c.completedOutputs = append(c.completedOutputs, responses.OutputMessageUnion{
		OfImageGenerationCall: &responses.ImageGenerationCallMessage{
			ID:           c.outputItemID,
			Status:       "completed",
			OutputFormat: strings.TrimPrefix(c.currentBlock.InlineData.MimeType, "image/"),
			Result:       c.accumulatedData,
			Background:   "",
			Quality:      "",
			Size:         "",
		},
	})

	return []*responses.ResponseChunk{
		c.buildOutputItemDoneImageGenerationCall(c.currentBlock.InlineData.MimeType, c.currentBlock.InlineData.Data),
	}
}

// =============================================================================
// Code Execution Handling
// =============================================================================

func (c *ResponseChunkToNativeResponseChunkConverter) handleExecutableCodePart(part *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// output_item.added
	out = append(out, c.buildOutputItemAddedCodeInterpreterCall(""))
	// code_interpreter_call.in_progress
	out = append(out, c.buildCodeInterpreterCallInProgress())
	// code_interpreter_call_code.delta
	out = append(out, c.buildCodeInterpreterCallCodeDelta(part.ExecutableCode.Code))
	// code_interpreter_call_code.done
	out = append(out, c.buildCodeInterpreterCallCodeDone(part.ExecutableCode.Code, part.ThoughtSignature))
	// code_interpreter_call.interpreting
	out = append(out, c.buildCodeInterpreterCallInterpreting())

	// Store the code
	c.accumulatedData = part.ExecutableCode.Code

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeExecutableCodePart() []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Store the code
	c.accumulatedData = c.currentBlock.ExecutableCode.Code

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) handleCodeExecutionResultPart(_ *Part) []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// Do nothing

	return out
}

func (c *ResponseChunkToNativeResponseChunkConverter) completeCodeExecutionResult() []*responses.ResponseChunk {
	var out []*responses.ResponseChunk

	// code_interpreter_call.completed
	out = append(out, c.buildCodeInterpreterCallCompleted(c.accumulatedData))

	// output_item.done
	out = append(out, c.buildOutputItemDoneCodeInterpreterCall(c.accumulatedData, c.previousPart.CodeExecutionResult.Output))

	return out
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
	c.outputItemID = responses.NewOutputItemMessageID()

	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:    "message",
				Id:      c.outputItemID,
				Status:  "in_progress",
				Role:    RoleModel.ToNativeRole(),
				Content: responses.OutputContent{},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedFunctionCall(callID, name, args string, thoughtSignature *string) *responses.ResponseChunk {
	c.outputItemID = responses.NewOutputItemFunctionCallID()

	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:             "function_call",
				Id:               c.outputItemID,
				Status:           "in_progress",
				CallID:           utils.Ptr(callID),
				Name:             utils.Ptr(name),
				Arguments:        utils.Ptr(args),
				ThoughtSignature: thoughtSignature, // Special case: gemini can have thought signature of function calls
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedReasoning() *responses.ResponseChunk {
	c.outputItemID = responses.NewOutputItemReasoningID()

	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    c.outputIndex,
			Item: responses.ChunkOutputItemData{
				Type:             "reasoning",
				Id:               c.outputItemID,
				Status:           "in_progress",
				Summary:          []responses.SummaryTextContent{},
				EncryptedContent: utils.Ptr(""),
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildContentPartAddedText() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfContentPartAdded: &responses.ChunkContentPart[constants.ChunkTypeContentPartAdded]{
			Type:           constants.ChunkTypeContentPartAdded(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Part:           responses.OutputContentUnion{OfOutputText: &responses.OutputTextContent{Text: ""}},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputTextDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputTextDelta: &responses.ChunkOutputText[constants.ChunkTypeOutputTextDelta]{
			Type:           constants.ChunkTypeOutputTextDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Delta:          delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildFunctionCallArgumentsDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfFunctionCallArgumentsDelta: &responses.ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDelta]{
			Type:           constants.ChunkTypeFunctionCallArgumentsDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			Delta:          delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputTextDone(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputTextDone: &responses.ChunkOutputText[constants.ChunkTypeOutputTextDone]{
			Type:           constants.ChunkTypeOutputTextDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
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
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			ContentIndex:   c.contentIndex,
			Part:           responses.OutputContentUnion{OfOutputText: &responses.OutputTextContent{Text: text}},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneMessage(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:    "message",
				Id:      c.outputItemID,
				Status:  "completed",
				Role:    RoleModel.ToNativeRole(),
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
			ItemId:         c.outputItemID,
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
				Type:             "function_call",
				Id:               c.outputItemID,
				Status:           "completed",
				CallID:           utils.Ptr(callID),
				Name:             utils.Ptr(name),
				Arguments:        utils.Ptr(args),
				ThoughtSignature: c.previousPart.ThoughtSignature,
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryPartAdded() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryPartAdded: &responses.ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartAdded]{
			Type:           constants.ChunkTypeReasoningSummaryPartAdded(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Part:           responses.SummaryTextContent{Text: ""},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryTextDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryTextDelta: &responses.ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDelta]{
			Type:           constants.ChunkTypeReasoningSummaryTextDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Delta:          delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildReasoningSummaryTextDone(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfReasoningSummaryTextDone: &responses.ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDone]{
			Type:           constants.ChunkTypeReasoningSummaryTextDone(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
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
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			SummaryIndex:   c.contentIndex,
			Part: responses.SummaryTextContent{
				Text: text,
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneReasoningSummary(text string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:    "reasoning",
				Id:      c.outputItemID,
				Status:  "completed",
				Summary: []responses.SummaryTextContent{{Text: text}},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedImageGenerationCall() *responses.ResponseChunk {
	c.outputItemID = responses.NewOutputItemReasoningID()

	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    c.outputIndex,
			Item: responses.ChunkOutputItemData{
				Type:   "image_generation_call",
				Id:     c.outputItemID,
				Status: "in_progress",

				Background:   utils.Ptr(""),
				Result:       utils.Ptr(""),
				Size:         utils.Ptr(""),
				OutputFormat: utils.Ptr(""),
				Quality:      utils.Ptr(""),
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildImageGenerationCallInProgress() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfImageGenerationCallInProgress: &responses.ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallInProgress]{
			Type:               constants.ChunkTypeImageGenerationCallInProgress(""),
			SequenceNumber:     c.nextSeqNum(),
			ItemId:             c.outputItemID,
			OutputIndex:        c.outputIndex,
			PartialImageIndex:  c.contentIndex,
			PartialImageBase64: "",
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildImageGenerationCallGenerating() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfImageGenerationCallGenerating: &responses.ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallGenerating]{
			Type:               constants.ChunkTypeImageGenerationCallGenerating(""),
			SequenceNumber:     c.nextSeqNum(),
			ItemId:             c.outputItemID,
			OutputIndex:        c.outputIndex,
			PartialImageIndex:  c.contentIndex,
			PartialImageBase64: "",
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildImageGenerationCallPartialImage(outputFormat string, imageData string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfImageGenerationCallPartialImage: &responses.ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallPartialImage]{
			Type:               constants.ChunkTypeImageGenerationCallPartialImage(""),
			SequenceNumber:     c.nextSeqNum(),
			ItemId:             c.outputItemID,
			OutputIndex:        c.outputIndex,
			PartialImageIndex:  c.contentIndex,
			PartialImageBase64: imageData,
			OutputFormat:       utils.Ptr(strings.TrimPrefix(outputFormat, "image/")),

			// Following cannot be mapped
			Background: utils.Ptr(""),
			Quality:    utils.Ptr(""),
			Size:       utils.Ptr(""),
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneImageGenerationCall(outputFormat string, imageData string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    0,
			Item: responses.ChunkOutputItemData{
				Type:   "image_generation_call",
				Id:     c.outputItemID,
				Status: "completed",

				Background:   utils.Ptr(""),
				Size:         utils.Ptr(""),
				Quality:      utils.Ptr(""),
				OutputFormat: utils.Ptr(strings.TrimPrefix(outputFormat, "image/")),
				Result:       utils.Ptr(imageData),
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildResponseCompleted() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfResponseCompleted: &responses.ChunkResponse[constants.ChunkTypeResponseCompleted]{
			Type:           constants.ChunkTypeResponseCompleted(""),
			SequenceNumber: c.nextSeqNum(),
			Response: responses.ChunkResponseData{
				Id:        c.messageID,
				Object:    "response",
				CreatedAt: int(time.Now().Unix()),
				Status:    "completed",
				Output:    c.completedOutputs,
				Usage: responses.Usage{
					InputTokens: c.usage.PromptTokenCount,
					InputTokensDetails: struct {
						CachedTokens int `json:"cached_tokens"`
					}{CachedTokens: 0},
					OutputTokens: c.usage.CandidatesTokenCount,
					TotalTokens:  c.usage.TotalTokenCount,
					OutputTokensDetails: struct {
						ReasoningTokens int `json:"reasoning_tokens"`
					}{ReasoningTokens: c.usage.ThoughtsTokenCount},
				},
				Request: responses.Request{Model: c.model},
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemAddedCodeInterpreterCall(code string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemAdded: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemAdded]{
			Type:           constants.ChunkTypeOutputItemAdded(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    c.outputIndex,
			Item: responses.ChunkOutputItemData{
				Type:   "code_interpreter_call",
				Id:     c.outputItemID,
				Status: "in_progress",
				Code:   &code,
			},
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildCodeInterpreterCallInProgress() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfCodeInterpreterCallInProgress: &responses.ChunkCodeInterpreterCall[constants.ChunkTypeCodeInterpreterCallInProgress]{
			Type:           constants.ChunkTypeCodeInterpreterCallInProgress(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildCodeInterpreterCallCodeDelta(delta string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfCodeInterpreterCallCodeDelta: &responses.ChunkCodeInterpreterCall[constants.ChunkTypeCodeInterpreterCallCodeDelta]{
			Type:           constants.ChunkTypeCodeInterpreterCallCodeDelta(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			Delta:          &delta,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildCodeInterpreterCallCodeDone(code string, thoughtSignature *string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfCodeInterpreterCallCodeDone: &responses.ChunkCodeInterpreterCall[constants.ChunkTypeCodeInterpreterCallCodeDone]{
			Type:             constants.ChunkTypeCodeInterpreterCallCodeDone(""),
			SequenceNumber:   c.nextSeqNum(),
			ItemId:           c.outputItemID,
			OutputIndex:      c.outputIndex,
			Code:             &code,
			ThoughtSignature: thoughtSignature,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildCodeInterpreterCallInterpreting() *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfCodeInterpreterCallInterpreting: &responses.ChunkCodeInterpreterCall[constants.ChunkTypeCodeInterpreterCallInterpreting]{
			Type:           constants.ChunkTypeCodeInterpreterCallInterpreting(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildCodeInterpreterCallCompleted(code string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfCodeInterpreterCallCompleted: &responses.ChunkCodeInterpreterCall[constants.ChunkTypeCodeInterpreterCallCompleted]{
			Type:           constants.ChunkTypeCodeInterpreterCallCompleted(""),
			SequenceNumber: c.nextSeqNum(),
			ItemId:         c.outputItemID,
			OutputIndex:    c.outputIndex,
			Code:           &code,
		},
	}
}

func (c *ResponseChunkToNativeResponseChunkConverter) buildOutputItemDoneCodeInterpreterCall(code string, output string) *responses.ResponseChunk {
	return &responses.ResponseChunk{
		OfOutputItemDone: &responses.ChunkOutputItem[constants.ChunkTypeOutputItemDone]{
			Type:           constants.ChunkTypeOutputItemDone(""),
			SequenceNumber: c.nextSeqNum(),
			OutputIndex:    c.outputIndex,
			Item: responses.ChunkOutputItemData{
				Type:    "code_interpreter_call",
				Id:      c.outputItemID,
				Status:  "completed",
				Code:    &code,
				Outputs: []responses.CodeInterpreterCallOutputParam{{Type: "logs", Logs: output}},
			},
		},
	}
}
