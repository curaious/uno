package main

import (
	"context"
	"fmt"
	"log"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway/sdk"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func main() {
	client, err := sdk.NewClient(&sdk.ClientOptions{
		LLMConfigs: adapters.NewInMemoryConfigStore([]*adapters.ProviderConfig{
			{
				ProviderName:  llm.ProviderNameOpenAI,
				BaseURL:       "",
				CustomHeaders: nil,
				Keys: []*adapters.ProviderKey{
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

	for chunk := range stream {
		switch chunk.ChunkType() {
		case "response.output_item.done":
			if chunk.OfOutputItemDone.Item.Type == "reasoning" {
				reasoning := &responses.ReasoningMessage{
					Type:             "",
					ID:               chunk.OfOutputItemDone.Item.Id,
					Summary:          chunk.OfOutputItemDone.Item.Summary,
					EncryptedContent: chunk.OfOutputItemDone.Item.EncryptedContent,
				}

				fmt.Println(reasoning.Summary)
			}
		}
	}
}
