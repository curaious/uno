package main

import (
	"context"
	"fmt"
	"log"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
)

func main() {
	client, err := sdk.New(&sdk.ClientOptions{
		Endpoint:    "http://localhost:6060",
		ProjectName: "Planner3",
		VirtualKey:  "sk-amg-qi590eZbIv9MQym__ww__31AGBIo6M_9AfSuI6fxbfc",
	})
	if err != nil {
		log.Fatal(err)
	}

	agent := client.NewAgent(&agents.AgentOptions{
		Name:        "Hello world agent",
		Instruction: "You are helpful assistant. You greet user with a light-joke",
		LLM: client.NewLLM(sdk.LLMOptions{
			Provider: llm.ProviderNameOpenAI,
			Model:    "gpt-4o-mini",
		}),
		Parameters: responses.Parameters{
			Temperature: utils.Ptr(0.2),
		},
	})

	out, err := agent.Execute(context.Background(), []responses.InputMessageUnion{
		responses.UserMessage("Hello!"),
	}, core.NilCallback)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(out[0].OfOutputMessage.Content[0].OfOutputText.Text)
}
