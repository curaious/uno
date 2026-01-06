package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/curaious/uno/pkg/sdk"
)

func main() {
	client, err := sdk.New(&sdk.ClientOptions{
		LLMConfigs: sdk.NewInMemoryConfigStore([]*gateway.ProviderConfig{
			{
				ProviderName:  llm.ProviderNameOpenAI,
				BaseURL:       "",
				CustomHeaders: nil,
				ApiKeys: []*gateway.APIKeyConfig{
					{
						Name:   "Key 1",
						APIKey: os.Getenv("OPENAI_API_KEY"),
					},
				},
			},
		}),
	})

	if err != nil {
		log.Fatal(err)
	}

	model := client.NewLLM(sdk.LLMOptions{
		Provider: llm.ProviderNameOpenAI,
		Model:    "o4-mini",
	})

	stream, err := model.NewStreamingResponses(
		context.Background(),
		&responses.Request{
			Instructions: utils.Ptr("You are helpful assistant. Reason before answering."),
			Input: responses.InputUnion{
				OfString: utils.Ptr("If 2+4=6, what would be 22+44=?"),
			},
			Parameters: responses.Parameters{
				Reasoning: &responses.ReasoningParam{
					Summary: utils.Ptr("detailed"),
				},
				Include: []responses.Includable{
					responses.IncludableReasoningEncryptedContent,
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	reasoningTxt := ""
	for chunk := range stream {
		switch chunk.ChunkType() {
		case "response.reasoning_summary_text.delta":
			reasoningTxt += chunk.OfReasoningSummaryTextDelta.Delta
		}
	}

	fmt.Println(reasoningTxt)
}
