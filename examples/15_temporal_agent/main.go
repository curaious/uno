package main

import (
	"log"
	"net/http"
	"os"

	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
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
		TemporalConfig: sdk.TemporalConfig{
			Endpoint: "0.0.0.0:7233",
		},
		RedisConfig: sdk.RedisConfig{
			Endpoint: "localhost:6379",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	model := client.NewLLM(sdk.LLMOptions{
		Provider: llm.ProviderNameOpenAI,
		Model:    "gpt-4.1-mini",
	})

	history := client.NewConversationManager()
	agentName := "SampleAgent"
	_ = client.NewTemporalAgent(&sdk.AgentOptions{
		Name:        agentName,
		Instruction: client.Prompt("You are helpful assistant. You are interacting with the user named {{name}}"),
		LLM:         model,
		History:     history,
	})

	client.StartTemporalService()              // Do this on the temporal service
	err = http.ListenAndServe(":8070", client) // Do this on the application that invokes the temporal workflow
	if err != nil {
		log.Fatal(err)
	}
}
