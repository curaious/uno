package gemini_speech

import (
	"encoding/base64"

	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway/providers/gemini/gemini_responses"
	"github.com/curaious/uno/pkg/llm/speech"
)

type Request struct {
	Model            string                     `json:"model"`
	Contents         []gemini_responses.Content `json:"contents"`
	GenerationConfig GenerationConfig           `json:"generationConfig"`
}

type GenerationConfig struct {
	ResponseModalities []string          `json:"responseModalities"` // "AUDIO"
	SpeechConfig       SpeechConfigParam `json:"speechConfig"`
}

type SpeechConfigParam struct {
	VoiceConfig VoiceConfigParam `json:"voiceConfig"`
}

type VoiceConfigParam struct {
	PrebuiltVoiceConfig PrebuiltVoiceConfig `json:"prebuiltVoiceConfig"`
}

type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

func (r *Request) ToNativeRequest() *speech.Request {
	return &speech.Request{
		Input: r.Contents[0].String(),
		Model: r.Model,
		Voice: r.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName,
	}
}

func NativeRequestToRequest(in *speech.Request) *Request {
	return &Request{
		Model: in.Model,
		Contents: []gemini_responses.Content{
			{
				Parts: []gemini_responses.Part{
					{
						Text: &in.Input,
					},
				},
			},
		},
		GenerationConfig: GenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: SpeechConfigParam{
				VoiceConfig: VoiceConfigParam{
					PrebuiltVoiceConfig: PrebuiltVoiceConfig{
						VoiceName: in.Voice,
					},
				},
			},
		},
	}
}

type Response struct {
	*gemini_responses.Response
}

func (r *Response) ToNativeResponse(responseFormat *string) *speech.Response {
	audioData := r.Candidates[0].Content.Parts[0].InlineData.Data

	audioBinaryData, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return nil
	}

	contentType := "audio/pcm; rate=24000; channels=1"
	if responseFormat != nil && *responseFormat == "wav" {
		contentType = "audio/wav"
		audioBinaryData, err = utils.Base64PCMToWAV(audioData, 24000, 1, 16)
		if err != nil {
			return nil
		}
	}

	return &speech.Response{
		Audio:       audioBinaryData,
		ContentType: contentType,
		Usage: speech.Usage{
			InputTokens: r.UsageMetadata.PromptTokenCount,
			InputTokensDetails: struct {
				CachedTokens int `json:"cached_tokens"`
			}{CachedTokens: 0},
			OutputTokens: r.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  r.UsageMetadata.TotalTokenCount,
			OutputTokensDetails: struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			}{ReasoningTokens: r.UsageMetadata.ThoughtsTokenCount},
		},
		RawFields: map[string]interface{}{
			"Gemini": r,
		},
	}
}

func (r *Response) ToNativeAudio(responseFormat *string) ([]byte, string, error) {
	audioData := r.Candidates[0].Content.Parts[0].InlineData.Data

	pcmData, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return nil, "", err
	}

	if responseFormat == nil {
		return pcmData, "audio/L16; codec=pcm; rate=24000; channels=1", nil
	}

	// convert pcm to wav
	switch *responseFormat {
	case "wav":
		wavData, err := utils.Base64PCMToWAV(audioData, 24000, 1, 16)
		if err != nil {
			return nil, "", err
		}
		return wavData, "audio/wav", nil

	default:
		return pcmData, "audio/pcm; rate=24000; channels=1", nil
	}
}

func NativeResponseToResponse(in *speech.Response) *Response {
	if raw, exists := in.RawFields["Gemini"]; exists {
		if geminiResp, ok := raw.(*Response); ok {
			return geminiResp
		}
	}

	return &Response{
		Response: &gemini_responses.Response{
			Candidates: []gemini_responses.Candidate{
				{
					Content: gemini_responses.Content{
						Parts: []gemini_responses.Part{
							{
								InlineData: &gemini_responses.InlinePartData{
									Data:     base64.StdEncoding.EncodeToString(in.Audio),
									MimeType: in.ContentType,
								},
							},
						},
					},
				},
			},
			UsageMetadata: &gemini_responses.UsageMetadata{
				PromptTokenCount:     in.Usage.InputTokens,
				CandidatesTokenCount: in.Usage.OutputTokens,
				TotalTokenCount:      in.Usage.TotalTokens,
				PromptTokensDetails:  nil,
				ThoughtsTokenCount:   in.Usage.OutputTokensDetails.ReasoningTokens,
			},
			ModelVersion: "",
			ResponseID:   "",
		},
	}
}

type NativeResponseChunkToResponseChunkConverter struct {
	audioData string
}

func (n *NativeResponseChunkToResponseChunkConverter) NativeResponseChunkToResponseChunk(in *speech.ResponseChunk) []Response {
	if in.OfAudioDelta != nil {
		n.audioData += in.OfAudioDelta.Audio
		return []Response{}
	}

	return []Response{
		{
			Response: &gemini_responses.Response{
				Candidates: []gemini_responses.Candidate{
					{
						Content: gemini_responses.Content{
							Parts: []gemini_responses.Part{
								{
									InlineData: &gemini_responses.InlinePartData{
										Data:     n.audioData,
										MimeType: "audio/pcm; rate=24000; channels=1",
									},
								},
							},
						},
					},
				},
				UsageMetadata: &gemini_responses.UsageMetadata{
					PromptTokenCount:     in.OfAudioDone.Usage.InputTokens,
					CandidatesTokenCount: in.OfAudioDone.Usage.OutputTokens,
					TotalTokenCount:      in.OfAudioDone.Usage.TotalTokens,
					PromptTokensDetails:  nil,
					ThoughtsTokenCount:   in.OfAudioDone.Usage.OutputTokensDetails.ReasoningTokens,
				},
				ModelVersion: "",
				ResponseID:   "",
			},
		},
	}
}

type ResponseChunkToNativeResponseChunkConverter struct {
	usageMetadata *gemini_responses.UsageMetadata
}

func (c *ResponseChunkToNativeResponseChunkConverter) ResponseChunkToNativeResponseChunk(in *Response) []*speech.ResponseChunk {
	if in == nil {
		return []*speech.ResponseChunk{
			{
				OfAudioDone: &speech.ChunkAudioDone[speech.ChunkTypeAudioDone]{
					Usage: speech.Usage{
						InputTokens: c.usageMetadata.PromptTokenCount,
						InputTokensDetails: struct {
							CachedTokens int `json:"cached_tokens"`
						}{CachedTokens: 0},
						OutputTokens: c.usageMetadata.CandidatesTokenCount,
						TotalTokens:  c.usageMetadata.TotalTokenCount,
						OutputTokensDetails: struct {
							ReasoningTokens int `json:"reasoning_tokens"`
						}{ReasoningTokens: c.usageMetadata.ThoughtsTokenCount},
					},
				},
			},
		}
	}

	c.usageMetadata = in.UsageMetadata

	return []*speech.ResponseChunk{
		{
			OfAudioDelta: &speech.ChunkAudioDelta[speech.ChunkTypeAudioDelta]{
				Audio: in.Candidates[0].Content.Parts[0].InlineData.Data,
			},
		},
	}
}
