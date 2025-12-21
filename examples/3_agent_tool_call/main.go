package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bytedance/sonic"
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
		Instruction: "You are helpful assistant. Use the get_user_name to get the user's name, and use it to greet the user.",
		LLM:         model,
		Tools: []core.Tool{
			NewCustomTool(),
		},
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

type CustomTool struct {
	*responses.ToolUnion
}

func NewCustomTool() *CustomTool {
	return &CustomTool{
		ToolUnion: &responses.ToolUnion{
			OfFunction: &responses.FunctionTool{
				Name:        "get_user_name",
				Description: utils.Ptr("Returns the user's name"),
			},
		},
	}
}

func (p *CustomTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr("world"),
		},
	}, nil
}
