package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway/sdk"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/constants"
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
		Model:    "gpt-image-1-mini",
	})

	stream, err := model.NewStreamingResponses(
		context.Background(),
		&responses.Request{
			Instructions: utils.Ptr("Describe this image"),
			Input: responses.InputUnion{
				OfInputMessageList: responses.InputMessageList{
					{
						OfEasyInput: &responses.EasyMessage{
							Role: constants.RoleUser,
							Content: responses.EasyInputContentUnion{
								OfString: utils.Ptr("Describe this image"),
							},
						},
					},
					{
						OfInputMessage: &responses.InputMessage{
							Role: constants.RoleUser,
							Content: responses.InputContent{
								{
									OfInputImage: &responses.InputImageContent{
										ImageURL: utils.Ptr("data:image/png;base64,ad12fas123dfa123s1dfas23112dfasd"),
										Detail:   "auto",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	for chunk := range stream {
		b, err := sonic.Marshal(chunk)
		if err != nil {
			slog.Warn("unable to marshal chunk", slog.Any("error", err))
			continue
		}

		fmt.Println(string(b))
	}
}
