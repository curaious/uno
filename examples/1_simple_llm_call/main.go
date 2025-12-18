package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func main() {
	client, err := sdk.NewClient(&sdk.ClientOptions{
		Endpoint:    "http://localhost:6060",
		ProjectName: "projectName",
		LLMConfigs: adapters.NewInMemoryConfigStore(map[llm.ProviderName]*adapters.ProviderOptions{
			llm.ProviderNameOpenAI: {
				APIKey:        "",
				BaseURL:       "",
				CustomHeaders: nil,
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
			Instructions: utils.Ptr("You are helpful assistant. You greet user with a light-joke"),
			Input: responses.InputUnion{
				OfString: utils.Ptr("Hello!"),
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
