package tools

import (
	"context"

	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/llm/responses"
)

type ImageGenerationTool struct {
	*core.BaseTool
}

func NewImageGenerationTool() *ImageGenerationTool {
	return &ImageGenerationTool{}
}

func (t *ImageGenerationTool) Execute(ctx context.Context, params *core.ToolCall) (*responses.FunctionCallOutputMessage, error) {
	return nil, nil
}

func (t *ImageGenerationTool) Tool(ctx context.Context) *responses.ToolUnion {
	return &responses.ToolUnion{OfImageGeneration: &responses.ImageGenerationTool{}}
}
