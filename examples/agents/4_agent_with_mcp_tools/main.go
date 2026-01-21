package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
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

	mcpClient, err := mcpclient.NewSSEClient(context.Background(), "http://localhost:9001/sse",
		mcpclient.WithHeaders(map[string]string{
			"token": "your-token",
		}),
		mcpclient.WithToolFilter("list_users"),
		mcpclient.WithApprovalRequiredTools("list_users"),
	)
	if err != nil {
		log.Fatal(err)
	}

	agent := client.NewAgent(&sdk.AgentOptions{
		Name:        "Hello world agent",
		Instruction: client.Prompt("You are helpful assistant."),
		LLM:         model,
		McpServers:  []agents.MCPToolset{mcpClient},
	})

	out, err := agent.Execute(context.Background(), &agents.AgentInput{
		Messages: []responses.InputMessageUnion{
			responses.UserMessage("Hello!"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	b, _ := sonic.Marshal(out)
	fmt.Println(string(b))
}
