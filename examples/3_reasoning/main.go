package main

import (
	"context"
	"fmt"
	"log"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
)

func main() {
	client, err := sdk.New(&sdk.ClientOptions{
		LLMConfigs: sdk.NewInMemoryConfigStore([]*sdk.ProviderConfig{
			{
				ProviderName:  llm.ProviderNameOpenAI,
				BaseURL:       "",
				CustomHeaders: nil,
				Keys: []*sdk.ProviderKey{
					{
						Name: "Key 1",
						Key:  "",
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
