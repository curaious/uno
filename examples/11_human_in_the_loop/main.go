package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/praveen001/uno/pkg/sdk"
)

// GetUserTool - runs immediately (no approval needed)
type GetUserTool struct {
	*core.BaseTool
}

func NewGetUserTool() *GetUserTool {
	return &GetUserTool{
		BaseTool: &core.BaseTool{
			RequiresApproval: false,
			ToolUnion: &responses.ToolUnion{
				OfFunction: &responses.FunctionTool{
					Name:        "get_user",
					Description: utils.Ptr("Gets user information"),
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"user_id": map[string]any{"type": "string"},
						},
						"required": []string{"user_id"},
					},
				},
			},
		},
	}
}

func (t *GetUserTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr(`{"name": "John Doe", "email": "john@example.com"}`),
		},
	}, nil
}

func (t *GetUserTool) Tool(ctx context.Context) *responses.ToolUnion { return t.ToolUnion }
func (t *GetUserTool) NeedApproval() bool                            { return t.RequiresApproval }

// DeleteUserTool - requires approval
type DeleteUserTool struct {
	*core.BaseTool
}

func NewDeleteUserTool() *DeleteUserTool {
	return &DeleteUserTool{
		BaseTool: &core.BaseTool{
			RequiresApproval: true, // Human approval required
			ToolUnion: &responses.ToolUnion{
				OfFunction: &responses.FunctionTool{
					Name:        "delete_user",
					Description: utils.Ptr("Permanently deletes a user account"),
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"user_id": map[string]any{"type": "string"},
						},
						"required": []string{"user_id"},
					},
				},
			},
		},
	}
}

func (t *DeleteUserTool) Execute(ctx context.Context, params *responses.FunctionCallMessage) (*responses.FunctionCallOutputMessage, error) {
	args := map[string]any{}
	json.Unmarshal([]byte(params.Arguments), &args)

	return &responses.FunctionCallOutputMessage{
		ID:     params.ID,
		CallID: params.CallID,
		Output: responses.FunctionCallOutputContentUnion{
			OfString: utils.Ptr(fmt.Sprintf("User %s has been deleted", args["user_id"])),
		},
	}, nil
}

func (t *DeleteUserTool) Tool(ctx context.Context) *responses.ToolUnion { return t.ToolUnion }
func (t *DeleteUserTool) NeedApproval() bool                            { return t.RequiresApproval }

func main() {
	ctx := context.Background()

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

	agent := client.NewAgent(&sdk.AgentOptions{
		Name:        "User Manager",
		Instruction: client.Prompt("You help manage user accounts."),
		LLM: client.NewLLM(sdk.LLMOptions{
			Provider: llm.ProviderNameOpenAI,
			Model:    "gpt-4o-mini",
		}),
		Tools: []core.Tool{
			NewGetUserTool(),
			NewDeleteUserTool(),
		},
		History: client.NewConversationManager("default", ""),
	})

	// First execution - agent may request to delete a user
	result, err := agent.Execute(ctx, []responses.InputMessageUnion{
		responses.UserMessage("Delete user 123"),
	}, func(chunk *responses.ResponseChunk) {
		// Handle streaming chunks
	})

	// Check if approval is needed
	if result.Status == core.RunStatusPaused {
		fmt.Println("Approval required for:", result.PendingApprovals)

		// Simulate user approval
		approvalResponse := responses.InputMessageUnion{
			OfFunctionCallApprovalResponse: &responses.FunctionCallApprovalResponseMessage{
				ID:              uuid.NewString(),
				ApprovedCallIds: []string{result.PendingApprovals[0].CallID},
				RejectedCallIds: []string{},
			},
		}

		// Resume with approval
		result, err = agent.Execute(ctx, []responses.InputMessageUnion{approvalResponse}, core.NilCallback)
	}

	if err != nil {
		log.Fatal(err)
	}

	buf, _ := sonic.Marshal(result.Output)
	fmt.Println("Final result:", string(buf))
}
