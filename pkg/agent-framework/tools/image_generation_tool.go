package tools

import (
	"context"

	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type ImageGenerationTool struct {
	*core.BaseTool
}

func NewImageGenerationTool() *ImageGenerationTool {
	return &ImageGenerationTool{}
}

func (t *ImageGenerationTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return nil, nil
}

func (t *ImageGenerationTool) Tool(ctx context.Context) *responses.ToolUnion {
	return &responses.ToolUnion{OfImageGeneration: &responses.ImageGenerationTool{}}
}
