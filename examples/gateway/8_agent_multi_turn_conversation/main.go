package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
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

	history := client.NewConversationManager("default", "")
	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        "Hello world agent",
		Instruction: "You are helpful assistant.",
		LLM:         model,
		History:     history,
	})

	out, err := agent.Execute(context.Background(), []responses.InputMessageUnion{
		responses.UserMessage("Hello! My name is Alice"),
	}, core.NilCallback)
	if err != nil {
		log.Fatal(err)
	}

	b, _ := sonic.Marshal(out)
	fmt.Println(string(b))

	// Agent itself is stateless - you can either re-create another agent or reuse the same agent instance
	// as long as the same history is given, it retains the context.
	agent2 := agents.NewAgent(&agents.AgentOptions{
		Name:        "Hello world agent",
		Instruction: "You are helpful assistant.",
		LLM:         model,
		History:     history,
	})

	out, err = agent2.Execute(context.Background(), []responses.InputMessageUnion{
		responses.UserMessage("What's my name?"),
	}, core.NilCallback)
	if err != nil {
		log.Fatal(err)
	}

	b, _ = sonic.Marshal(out)
	fmt.Println(string(b))
}
