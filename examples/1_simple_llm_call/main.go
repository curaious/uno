package main

import (
	"context"
	"fmt"
	"log"

	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
)

func main() {
	client, err := sdk.NewClient(&sdk.ClientOptions{
		Endpoint:    "http://localhost:6060",
		ProjectName: "Planner3",
		VirtualKey:  "",
	})
	if err != nil {
		log.Fatal(err)
	}

	model := client.NewLLM(sdk.LLMOptions{
		Provider: llm.ProviderNameOpenAI,
		Model:    "gpt-4o-mini",
	})

	resp, err := model.NewStreamingResponses(
		context.Background(),
		responses.Request{
			Instructions: utils.Ptr("You are helpful assistant. You greet user with a light-joke"),
			Input: responses.InputUnion{
				OfString: utils.Ptr("Hello!"),
			},
		},
		func(message *responses.ResponseChunk) {
			fmt.Println(message)
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp)
}
