package speech

type Response struct {
	Audio       []byte `json:"audio"`
	ContentType string `json:"content_type"`
}

type ResponseChunk struct {
	OfAudioDelta *ChunkAudioDelta[ChunkTypeAudioDelta] `json:",omitempty"`
	OfAudioDone  *ChunkAudioDone[ChunkTypeAudioDone]   `json:",omitempty"`
}

type ChunkAudioDelta[T any] struct {
	Type  T      `json:"type"`
	Audio string `json:"audio"`
}

type ChunkAudioDone[T any] struct {
	Type  T     `json:"type"`
	Usage Usage `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
