package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
)

func main() {
	client, err := sdk.NewClient(&sdk.ClientOptions{
		Endpoint:    "http://localhost:6060",
		ProjectName: "Planner3",
		VirtualKey:  "sk-amg-hEeK6NT00_c4BCHtZskguwZUjlqzqVoyBN4JhRnAtUg",
	})
	if err != nil {
		log.Fatal(err)
	}

	model := client.NewLLM(sdk.LLMOptions{
		Provider: llm.ProviderNameOpenAI,
		Model:    "gpt-4o-mini",
	})

	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        "Hello world agent",
		Instruction: "You are helpful assistant.",
		LLM:         model,
		History:     client.NewConversationManager("default", uuid.NewString(), ""),
	})

	out, err := agent.Execute(context.Background(), []responses.InputMessageUnion{
		responses.UserMessage("Hello!"),
	}, core.NilCallback)
	if err != nil {
		log.Fatal(err)
	}

	b, _ := sonic.Marshal(out)
	fmt.Println(string(b))

	out, err = agent.Execute(context.Background(), []responses.InputMessageUnion{
		responses.UserMessage("what was my previous message?"),
	}, core.NilCallback)
	if err != nil {
		log.Fatal(err)
	}

	b, _ = sonic.Marshal(out)
	fmt.Println(string(b))
}
