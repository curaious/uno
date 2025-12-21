package main

import (
	"context"
	"fmt"
	"log"

	json "github.com/bytedance/sonic"
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
		Model:    "gpt-4.1-mini",
	})

	stream, err := model.NewStreamingResponses(
		context.Background(),
		&responses.Request{
			Instructions: utils.Ptr("You are helpful assistant. You will greet the user by their name."),
			Input: responses.InputUnion{
				OfString: utils.Ptr("Hello!"),
			},
			Tools: []responses.ToolUnion{
				{
					OfFunction: &responses.FunctionTool{
						Name:        "get_user_name",
						Description: utils.Ptr("This tool returns the user's name"),
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	var fnCalls []*responses.FunctionCallMessage
	for chunk := range stream {
		switch chunk.ChunkType() {
		case "response.output_item.done":
			if chunk.OfOutputItemDone.Item.Type == "function_call" {
				fnCall := &responses.FunctionCallMessage{
					ID:        chunk.OfOutputItemDone.Item.Id,
					CallID:    *chunk.OfOutputItemDone.Item.CallID,
					Name:      *chunk.OfOutputItemDone.Item.Name,
					Arguments: *chunk.OfOutputItemDone.Item.Arguments,
				}
				fnCalls = append(fnCalls, fnCall)
			}
		}
	}

	// Handle function calls
	t, _ := json.Marshal(fnCalls)
	fmt.Println(string(t))
}
