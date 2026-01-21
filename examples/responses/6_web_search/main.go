package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/constants"
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
		Model:    "gpt-4.1-mini",
	})

	stream, err := model.NewStreamingResponses(
		context.Background(),
		&responses.Request{
			Input: responses.InputUnion{
				OfInputMessageList: responses.InputMessageList{
					{
						OfEasyInput: &responses.EasyMessage{
							Role: constants.RoleUser,
							Content: responses.EasyInputContentUnion{
								OfString: utils.Ptr("What are the latest AI news?"),
							},
						},
					},
				},
			},
			Tools: []responses.ToolUnion{
				{
					OfWebSearch: &responses.WebSearchTool{},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	for chunk := range stream {
		if chunk.OfOutputTextAnnotationAdded != nil {
			fmt.Println(chunk.OfOutputTextAnnotationAdded.Annotation)
		}

		if chunk.OfOutputItemDone != nil {
			if chunk.OfOutputItemDone.Item.Type == "web_search_call" {
				fmt.Println(chunk.OfOutputItemDone.Item.Action.OfSearch)
			}
		}
	}
}
