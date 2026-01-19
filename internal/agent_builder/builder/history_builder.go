package builder

import (
	"github.com/curaious/uno/internal/adapters"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/agent-framework/summariser"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	adapters2 "github.com/curaious/uno/pkg/sdk/adapters"
	"github.com/google/uuid"
)

func BuildConversationManager(svc *services.Services, projectID uuid.UUID, llmGateway *gateway.LLMGateway, config *agent_config.HistoryConfig, key string) (*history.CommonConversationManager, error) {
	if config == nil || !config.Enabled {
		return history.NewConversationManager(adapters2.NewInMemoryConversationPersistence()), nil
	}

	var options []history.ConversationManagerOptions
	if config.Summarizer != nil && config.Summarizer.Type != "none" {
		switch config.Summarizer.Type {
		case "llm":
			summarizerInstruction := BuildPrompt(svc.Prompt, projectID, config.Summarizer.LLMSummarizerPrompt)
			summarizerLLM := BuildLLMClient(llmGateway, key, llm.ProviderName(config.Summarizer.LLMSummarizerModel.ProviderType), config.Summarizer.LLMSummarizerModel.ModelID)
			summarizerModelParams, err := BuildModelParams(config.Summarizer.LLMSummarizerModel)
			if err != nil {
				return nil, err
			}

			summarizer := summariser.NewLLMHistorySummarizer(&summariser.LLMHistorySummarizerOptions{
				LLM:             summarizerLLM,
				Instruction:     summarizerInstruction,
				TokenThreshold:  *config.Summarizer.LLMTokenThreshold,
				KeepRecentCount: *config.Summarizer.LLMKeepRecentCount,
				Parameters:      summarizerModelParams,
			})
			options = append(options, history.WithSummarizer(summarizer))
		case "sliding_window":
			summarizer := summariser.NewSlidingWindowHistorySummarizer(&summariser.SlidingWindowHistorySummarizerOptions{
				KeepCount: *config.Summarizer.SlidingWindowKeepCount,
			})
			options = append(options, history.WithSummarizer(summarizer))
		}
	}

	return history.NewConversationManager(
		adapters.NewInternalConversationPersistence(svc.Conversation, projectID),
		options...,
	), nil
}
