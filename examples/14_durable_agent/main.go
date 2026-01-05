package main

import (
	"log"
	"net/http"
	"os"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/sdk"
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
		RestateConfig: sdk.RestateConfig{
			Endpoint: "http://localhost:8081",
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
	_ = client.NewRestateAgent(&sdk.AgentOptions{
		Name:        agentName,
		Instruction: client.Prompt("You are helpful assistant. You are interacting with the user named {{name}}"),
		LLM:         model,
		History:     history,
	})

	client.StartRestateService("0.0.0.0", "9081") // Do this on the restate service
	err = http.ListenAndServe(":8070", client)    // Do this on the application that invokes the restate workflow
	if err != nil {
		log.Fatal(err)
	}
}
