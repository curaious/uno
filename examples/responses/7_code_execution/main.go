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
			Instructions: utils.Ptr("You are a personal math tutor. When asked a math question, write and run code using the python tool to answer the question."),
			Input: responses.InputUnion{
				OfInputMessageList: responses.InputMessageList{
					{
						OfEasyInput: &responses.EasyMessage{
							Role: constants.RoleUser,
							Content: responses.EasyInputContentUnion{
								OfString: utils.Ptr("I need to solve the equation 3x + 11 = 14. Can you help me?"),
							},
						},
					},
				},
			},
			Tools: []responses.ToolUnion{
				{
					OfCodeExecution: &responses.CodeExecutionTool{
						Container: &responses.CodeExecutionToolContainerUnion{
							ContainerConfig: &responses.CodeExecutionToolContainerConfig{
								Type:        "auto",
								MemoryLimit: "4g",
							},
						},
					},
				},
			},
			Parameters: responses.Parameters{
				Include: []responses.Includable{
					responses.IncludableCodeInterpreterCallOutputs,
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	for chunk := range stream {
		if chunk.OfOutputItemDone != nil {
			if chunk.OfOutputItemDone.Item.Type == "code_interpreter_call" {
				fmt.Println(chunk.OfOutputItemDone.Item.Code, chunk.OfOutputItemDone.Item.Outputs)
			}
		}
	}
}
