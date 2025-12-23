package core

import (
	"context"
)

type SystemPromptProvider interface {
	GetPrompt(ctx context.Context) (string, error)
}
