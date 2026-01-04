package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
)

type CustomTool struct {
	*core.BaseTool
}

func NewCustomTool() *CustomTool {
	return &CustomTool{
		BaseTool: &core.BaseTool{
			ToolUnion: &responses.ToolUnion{
				OfFunction: &responses.FunctionTool{
					Name:        "get_user_name",
					Description: utils.Ptr("Returns the user's name"),
				},
			},
		},
	}
}

func (t *CustomTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr("Bob"),
		},
	}, nil
}

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

	history := client.NewConversationManager()
	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        "Hello world agent",
		Instruction: client.Prompt("You are helpful assistant. Greet the user by their name."),
		LLM:         model,
		History:     history,
		Tools: []core.Tool{
			NewCustomTool(),
		},
	})

	out, err := agent.Execute(context.Background(), &agents.AgentInput{
		Messages: []responses.InputMessageUnion{
			responses.UserMessage("Hello!"),
		},
		Namespace:         "default",
		PreviousMessageID: "",
	})
	if err != nil {
		log.Fatal(err)
	}

	b, _ := sonic.Marshal(out)
	fmt.Println(string(b))
}
