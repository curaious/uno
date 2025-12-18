package responses

import (
	"errors"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/pkg/llm/constants"
)

// --------------------//
// Chunk definitions //
// -----------------//

type ResponseChunk struct {
	OfResponseCreated    *ChunkResponse[constants.ChunkTypeResponseCreated]    `json:",omitempty"`
	OfResponseInProgress *ChunkResponse[constants.ChunkTypeResponseInProgress] `json:",omitempty"`
	OfResponseCompleted  *ChunkResponse[constants.ChunkTypeResponseCompleted]  `json:",omitempty"`

	OfOutputItemAdded *ChunkOutputItem[constants.ChunkTypeOutputItemAdded] `json:",omitempty"`
	OfOutputItemDone  *ChunkOutputItem[constants.ChunkTypeOutputItemDone]  `json:",omitempty"`

	// For output item of type "message"
	OfContentPartAdded *ChunkContentPart[constants.ChunkTypeContentPartAdded] `json:",omitempty"`
	OfContentPartDone  *ChunkContentPart[constants.ChunkTypeContentPartDone]  `json:",omitempty"`
	OfOutputTextDelta  *ChunkOutputText[constants.ChunkTypeOutputTextDelta]   `json:",omitempty"`
	OfOutputTextDone   *ChunkOutputText[constants.ChunkTypeOutputTextDone]    `json:",omitempty"`

	// For output item of type "function_call"
	OfFunctionCallArgumentsDelta *ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDelta] `json:",omitempty"`
	OfFunctionCallArgumentsDone  *ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDone]  `json:",omitempty"`

	// For output item of type "reasoning"
	OfReasoningSummaryPartAdded *ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartAdded] `json:",omitempty"`
	OfReasoningSummaryPartDone  *ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartDone]  `json:",omitempty"`
	OfReasoningSummaryTextDelta *ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDelta] `json:",omitempty"`
	OfReasoningSummaryTextDone  *ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDone]  `json:",omitempty"`

	// For output item of type "image_generation_call"
	OfImageGenerationCallInProgress   *ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallInProgress]   `json:",omitempty"`
	OfImageGenerationCallGenerating   *ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallGenerating]   `json:",omitempty"`
	OfImageGenerationCallPartialImage *ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallPartialImage] `json:",omitempty"`

	// Custom Chunks
	OfRunCreated         *ChunkResponse[constants.ChunkTypeRunCreated]   `json:",omitempty"`
	OfRunCompleted       *ChunkResponse[constants.ChunkTypeRunCompleted] `json:",omitempty"`
	OfFunctionCallOutput *FunctionCallOutputMessage                      `json:",omitempty"`
}

func (u *ResponseChunk) UnmarshalJSON(data []byte) error {
	var responseCreated *ChunkResponse[constants.ChunkTypeResponseCreated]
	if err := sonic.Unmarshal(data, &responseCreated); err == nil {
		u.OfResponseCreated = responseCreated
		return nil
	}

	var responseInProgress *ChunkResponse[constants.ChunkTypeResponseInProgress]
	if err := sonic.Unmarshal(data, &responseInProgress); err == nil {
		u.OfResponseInProgress = responseInProgress
		return nil
	}

	var responseCompleted *ChunkResponse[constants.ChunkTypeResponseCompleted]
	if err := sonic.Unmarshal(data, &responseCompleted); err == nil {
		u.OfResponseCompleted = responseCompleted
		return nil
	}

	var outputItemAdded *ChunkOutputItem[constants.ChunkTypeOutputItemAdded]
	if err := sonic.Unmarshal(data, &outputItemAdded); err == nil {
		u.OfOutputItemAdded = outputItemAdded
		return nil
	}

	var outputItemDone *ChunkOutputItem[constants.ChunkTypeOutputItemDone]
	if err := sonic.Unmarshal(data, &outputItemDone); err == nil {
		u.OfOutputItemDone = outputItemDone
		return nil
	}

	var contentPartAdded *ChunkContentPart[constants.ChunkTypeContentPartAdded]
	if err := sonic.Unmarshal(data, &contentPartAdded); err == nil {
		u.OfContentPartAdded = contentPartAdded
		return nil
	}

	var contentPartDone *ChunkContentPart[constants.ChunkTypeContentPartDone]
	if err := sonic.Unmarshal(data, &contentPartDone); err == nil {
		u.OfContentPartDone = contentPartDone
		return nil
	}

	var outputTextDelta *ChunkOutputText[constants.ChunkTypeOutputTextDelta]
	if err := sonic.Unmarshal(data, &outputTextDelta); err == nil {
		u.OfOutputTextDelta = outputTextDelta
		return nil
	}

	var outputTextDone *ChunkOutputText[constants.ChunkTypeOutputTextDone]
	if err := sonic.Unmarshal(data, &outputTextDone); err == nil {
		u.OfOutputTextDone = outputTextDone
		return nil
	}

	var fnCall *ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDelta]
	if err := sonic.Unmarshal(data, &fnCall); err == nil {
		u.OfFunctionCallArgumentsDelta = fnCall
		return nil
	}

	var fnCallDone *ChunkFunctionCall[constants.ChunkTypeFunctionCallArgumentsDone]
	if err := sonic.Unmarshal(data, &fnCallDone); err == nil {
		u.OfFunctionCallArgumentsDone = fnCallDone
		return nil
	}

	var reasoningSummaryPartAdded *ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartAdded]
	if err := sonic.Unmarshal(data, &reasoningSummaryPartAdded); err == nil {
		u.OfReasoningSummaryPartAdded = reasoningSummaryPartAdded
		return nil
	}

	var reasoningSummaryPartDone *ChunkReasoningSummaryPart[constants.ChunkTypeReasoningSummaryPartDone]
	if err := sonic.Unmarshal(data, &reasoningSummaryPartDone); err == nil {
		u.OfReasoningSummaryPartDone = reasoningSummaryPartDone
		return nil
	}

	var reasoningSummaryTextDelta *ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDelta]
	if err := sonic.Unmarshal(data, &reasoningSummaryTextDelta); err == nil {
		u.OfReasoningSummaryTextDelta = reasoningSummaryTextDelta
		return nil
	}

	var reasoningSummaryTextDone *ChunkReasoningSummaryText[constants.ChunkTypeReasoningSummaryTextDone]
	if err := sonic.Unmarshal(data, &reasoningSummaryTextDone); err == nil {
		u.OfReasoningSummaryTextDone = reasoningSummaryTextDone
		return nil
	}

	var imageGenerationCallInProgress *ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallInProgress]
	if err := sonic.Unmarshal(data, &imageGenerationCallInProgress); err == nil {
		u.OfImageGenerationCallInProgress = imageGenerationCallInProgress
		return nil
	}

	var imageGenerationCallGenerating *ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallGenerating]
	if err := sonic.Unmarshal(data, &imageGenerationCallGenerating); err == nil {
		u.OfImageGenerationCallGenerating = imageGenerationCallGenerating
		return nil
	}

	var imageGenerationCallPartialImage *ChunkImageGenerationCall[constants.ChunkTypeImageGenerationCallPartialImage]
	if err := sonic.Unmarshal(data, &imageGenerationCallPartialImage); err == nil {
		u.OfImageGenerationCallPartialImage = imageGenerationCallPartialImage
		return nil
	}

	return errors.New("invalid response chunk union")
}

func (u *ResponseChunk) MarshalJSON() ([]byte, error) {
	if u.OfResponseCreated != nil {
		return sonic.Marshal(u.OfResponseCreated)
	}

	if u.OfResponseInProgress != nil {
		return sonic.Marshal(u.OfResponseInProgress)
	}

	if u.OfResponseCompleted != nil {
		return sonic.Marshal(u.OfResponseCompleted)
	}

	if u.OfOutputItemAdded != nil {
		return sonic.Marshal(u.OfOutputItemAdded)
	}

	if u.OfOutputItemDone != nil {
		return sonic.Marshal(u.OfOutputItemDone)
	}

	if u.OfContentPartAdded != nil {
		return sonic.Marshal(u.OfContentPartAdded)
	}

	if u.OfContentPartDone != nil {
		return sonic.Marshal(u.OfContentPartDone)
	}

	if u.OfOutputTextDelta != nil {
		return sonic.Marshal(u.OfOutputTextDelta)
	}

	if u.OfOutputTextDone != nil {
		return sonic.Marshal(u.OfOutputTextDone)
	}

	if u.OfFunctionCallArgumentsDelta != nil {
		return sonic.Marshal(u.OfFunctionCallArgumentsDelta)
	}

	if u.OfFunctionCallArgumentsDone != nil {
		return sonic.Marshal(u.OfFunctionCallArgumentsDone)
	}

	if u.OfReasoningSummaryPartAdded != nil {
		return sonic.Marshal(u.OfReasoningSummaryPartAdded)
	}

	if u.OfReasoningSummaryPartDone != nil {
		return sonic.Marshal(u.OfReasoningSummaryPartDone)
	}

	if u.OfReasoningSummaryTextDelta != nil {
		return sonic.Marshal(u.OfReasoningSummaryTextDelta)
	}

	if u.OfImageGenerationCallInProgress != nil {
		return sonic.Marshal(u.OfImageGenerationCallInProgress)
	}

	if u.OfImageGenerationCallGenerating != nil {
		return sonic.Marshal(u.OfImageGenerationCallGenerating)
	}

	if u.OfImageGenerationCallPartialImage != nil {
		return sonic.Marshal(u.OfImageGenerationCallPartialImage)
	}

	if u.OfReasoningSummaryTextDone != nil {
		return sonic.Marshal(u.OfReasoningSummaryTextDone)
	}

	// Custom Chunks
	if u.OfRunCreated != nil {
		return sonic.Marshal(u.OfRunCreated)
	}

	if u.OfRunCompleted != nil {
		return sonic.Marshal(u.OfRunCompleted)
	}

	if u.OfFunctionCallOutput != nil {
		return sonic.Marshal(u.OfFunctionCallOutput)
	}

	return nil, nil
}

func (u *ResponseChunk) ChunkType() string {
	if u.OfResponseCreated != nil {
		return u.OfResponseCreated.Type.Value()
	}

	if u.OfResponseInProgress != nil {
		return u.OfResponseInProgress.Type.Value()
	}

	if u.OfResponseCompleted != nil {
		return u.OfResponseCompleted.Type.Value()
	}

	if u.OfOutputItemAdded != nil {
		return u.OfOutputItemAdded.Type.Value()
	}

	if u.OfOutputItemDone != nil {
		return u.OfOutputItemDone.Type.Value()
	}

	if u.OfContentPartAdded != nil {
		return u.OfContentPartAdded.Type.Value()
	}

	if u.OfContentPartDone != nil {
		return u.OfContentPartDone.Type.Value()
	}

	if u.OfOutputTextDelta != nil {
		return u.OfOutputTextDelta.Type.Value()
	}

	if u.OfOutputTextDone != nil {
		return u.OfOutputTextDone.Type.Value()
	}

	if u.OfFunctionCallArgumentsDelta != nil {
		return u.OfFunctionCallArgumentsDelta.Type.Value()
	}

	if u.OfFunctionCallArgumentsDone != nil {
		return u.OfFunctionCallArgumentsDone.Type.Value()
	}

	if u.OfReasoningSummaryPartAdded != nil {
		return u.OfReasoningSummaryPartAdded.Type.Value()
	}

	if u.OfReasoningSummaryPartDone != nil {
		return u.OfReasoningSummaryPartDone.Type.Value()
	}

	if u.OfReasoningSummaryTextDelta != nil {
		return u.OfReasoningSummaryTextDelta.Type.Value()
	}

	if u.OfReasoningSummaryTextDone != nil {
		return u.OfReasoningSummaryTextDone.Type.Value()
	}

	if u.OfImageGenerationCallInProgress != nil {
		return u.OfImageGenerationCallInProgress.Type.Value()
	}

	if u.OfImageGenerationCallGenerating != nil {
		return u.OfImageGenerationCallGenerating.Type.Value()
	}

	if u.OfImageGenerationCallPartialImage != nil {
		return u.OfImageGenerationCallPartialImage.Type.Value()
	}

	// Custom Chunks
	if u.OfRunCreated != nil {
		return u.OfRunCreated.Type.Value()
	}

	if u.OfRunCompleted != nil {
		return u.OfRunCompleted.Type.Value()
	}

	if u.OfFunctionCallOutput != nil {
		return u.OfFunctionCallOutput.Type.Value()
	}

	return ""
}

type ChunkResponse[T any] struct {
	Type           T                 `json:"type"`
	SequenceNumber int               `json:"sequence_number"`
	Response       ChunkResponseData `json:"response"`
}

type ChunkResponseData struct {
	Id                string               `json:"id"`
	Object            string               `json:"object"`
	CreatedAt         int                  `json:"created_at"`
	Status            string               `json:"status"`
	Background        bool                 `json:"background"`
	Error             interface{}          `json:"error"`
	IncompleteDetails interface{}          `json:"incomplete_details"`
	Output            []OutputMessageUnion `json:"output"`
	Usage             Usage                `json:"usage"`
	Request
}

type ChunkOutputItem[T any] struct {
	Type           T                   `json:"type"`
	SequenceNumber int                 `json:"sequence_number"`
	OutputIndex    int                 `json:"output_index"`
	Item           ChunkOutputItemData `json:"item"`
}

type ChunkOutputItemData struct {
	Type string `json:"type"` // "function_call" , "message", "reasoning", "image_generation_call"

	// Common fields
	Id     string `json:"id"`
	Status string `json:"status"`

	// For output_item of type "message"
	Content OutputContent  `json:"content"`
	Role    constants.Role `json:"role"`

	// For output_item of type "function_call"
	CallID    *string `json:"call_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	Arguments *string `json:"arguments,omitempty"`

	// For "reasoning"
	EncryptedContent *string              `json:"encrypted_content,omitempty"`
	Summary          []SummaryTextContent `json:"summary,omitempty"`

	// For "image_generation_call"
	Background   *string `json:"background,omitempty"`    // "opaque"
	OutputFormat *string `json:"output_format,omitempty"` // "png"
	Quality      *string `json:"quality,omitempty"`       // "medium"
	Result       *string `json:"result,omitempty"`        // base64 image
	Size         *string `json:"size,omitempty"`          // "1024x1024"
}

type ChunkContentPart[T any] struct {
	Type           T                  `json:"type"`
	SequenceNumber int                `json:"sequence_number"`
	ItemId         string             `json:"item_id"`
	OutputIndex    int                `json:"output_index"`
	ContentIndex   int                `json:"content_index"`
	Part           OutputContentUnion `json:"part"`
}

type ChunkOutputText[T any] struct {
	Type           T             `json:"type"`
	SequenceNumber int           `json:"sequence_number"`
	ItemId         string        `json:"item_id"`
	OutputIndex    int           `json:"output_index"`
	ContentIndex   int           `json:"content_index"`
	Delta          string        `json:"delta"`
	Logprobs       []interface{} `json:"logprobs"`
	Obfuscation    string        `json:"obfuscation"`

	// Only on content.output_text.done (contains the accumulated content)
	Text *string `json:"text,omitempty"`
}

type ChunkFunctionCall[T any] struct {
	Type T `json:"type"`

	SequenceNumber int    `json:"sequence_number"`
	ItemId         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	Delta          string `json:"delta"`
	Arguments      string `json:"arguments"`
	Obfuscation    string `json:"obfuscation"`
}

type ChunkReasoningSummaryPart[T any] struct {
	Type           T                  `json:"type"`
	SequenceNumber int                `json:"sequence_number"`
	ItemId         string             `json:"item_id"`
	OutputIndex    int                `json:"output_index"`
	SummaryIndex   int                `json:"summary_index"`
	Part           SummaryTextContent `json:"part"`
}

type ChunkReasoningSummaryText[T any] struct {
	Type           T      `json:"type"`
	SequenceNumber int    `json:"sequence_number"`
	ItemId         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`
	SummaryIndex   int    `json:"summary_index"`

	// Only on response.reasoning_summary_text.delta
	Delta string `json:"delta"`

	// Only on response.reasoning_summary_text.done
	Text *string `json:"text,omitempty"`

	// Helpers - Anthropic sends signature as a separate delta with different type "signature_delta"
	// We would have been using "Delta" for storing the reasoning summary delta, and since we need another field
	// to store the signature, we use this field.
	// We don't use "Text" to avoid confusion.
	EncryptedContent *string `json:"encrypted_content,omitempty"`
}

type ChunkImageGenerationCall[T any] struct {
	Type T `json:"type"`

	SequenceNumber int    `json:"sequence_number"`
	ItemId         string `json:"item_id"`
	OutputIndex    int    `json:"output_index"`

	// Only on response.image_generation_call.partial_image
	PartialImageIndex  int     `json:"partial_image_index"`
	PartialImageBase64 string  `json:"partial_image_b64"`
	Background         *string `json:"background,omitempty"`    // "opaque"
	OutputFormat       *string `json:"output_format,omitempty"` // "png"
	Quality            *string `json:"quality,omitempty"`       // "medium"
	Size               *string `json:"size,omitempty"`          // "1024x1024"
}

type ChunkResponseUsage struct {
	InputTokens        int `json:"input_tokens"`
	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputTokens        int `json:"output_tokens"`
	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
	TotalTokens int `json:"total_tokens"`
}
