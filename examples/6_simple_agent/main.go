package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
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
	})
	if err != nil {
		log.Fatal(err)
	}

	model := client.NewLLM(sdk.LLMOptions{
		Provider: llm.ProviderNameOpenAI,
		Model:    "gpt-4.1-mini",
	})

	history := client.NewConversationManager("default", "")
	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        "Hello world agent",
		Instruction: client.Prompt("You are helpful assistant. You are interacting with the user named {{name}}"),
		LLM:         model,
		History:     history,
	})

	out, err := agent.Execute(context.Background(), &agents.AgentInput{
		Messages: []responses.InputMessageUnion{
			responses.UserMessage("Hello!"),
		},
		RunContext: map[string]any{
			"name": "Bob",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	b, _ := sonic.Marshal(out)
	fmt.Println(string(b))
}
