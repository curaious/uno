package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/tools"
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

	mcpClient, err := tools.NewMCPServer(context.Background(), "http://localhost:9001/sse", nil)
	if err != nil {
		log.Fatal(err)
	}

	agent := client.NewAgent(&sdk.AgentOptions{
		Name:        "Hello world agent",
		Instruction: client.Prompt("You are helpful assistant."),
		LLM:         model,
		Tools:       mcpClient.GetTools(),
	})

	out, err := agent.Execute(context.Background(), []responses.InputMessageUnion{
		responses.UserMessage("Hello!"),
	}, core.NilCallback)
	if err != nil {
		log.Fatal(err)
	}

	b, _ := sonic.Marshal(out)
	fmt.Println(string(b))
}
