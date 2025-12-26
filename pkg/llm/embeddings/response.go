package embeddings

type Response struct {
	Object string          `json:"object"`
	Model  string          `json:"model"`
	Usage  *Usage          `json:"usage"`
	Data   []EmbeddingData `json:"data"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type Usage struct {
	PromptTokens int64 `json:"prompt_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}
