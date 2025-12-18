package sdk

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services/project"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/gateway/client"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

type Client struct {
	endpoint   string
	projectId  uuid.UUID
	virtualKey string
}

type ClientOptions struct {
	Endpoint    string
	ProjectName string
	VirtualKey  string
}

func NewClient(opts *ClientOptions) (*Client, error) {
	// Convert project name to ID
	url := fmt.Sprintf("%s/api/agent-server/projects", opts.Endpoint)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	projectsRes := adapters.Response[[]project.Project]{}
	if err := utils.DecodeJSON(resp.Body, &projectsRes); err != nil {
		return nil, err
	}

	for _, proj := range projectsRes.Data {
		if proj.Name == opts.ProjectName {
			return &Client{
				endpoint:   opts.Endpoint,
				projectId:  proj.ID,
				virtualKey: opts.VirtualKey,
			}, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", opts.ProjectName)
}

func (c *Client) NewConversationManager(namespace, msgId, previousMsgId string, opts ...history.ConversationManagerOptions) core.ChatHistory {
	return history.NewConversationManager(
		adapters.NewExternalConversationPersistence(c.endpoint),
		c.projectId,
		namespace,
		msgId,
		previousMsgId,
		opts...,
	)
}

func (c *Client) NewPromptManager(name string, label string, resolver core.SystemPromptResolver) core.SystemPromptProvider {
	return prompts.NewPromptManager(
		adapters.NewExternalPromptPersistence(c.endpoint, c.projectId),
		name,
		label,
		resolver,
	)
}

type LLMOptions struct {
	Provider llm.ProviderName
	Model    string
}

// NewLLM creates a new core.LLM that calls the agent-server via HTTP.
// Uses the ExternalLLMGateway which routes through the gateway API.
func (c *Client) NewLLM(opts LLMOptions) llm.Provider {
	return client.NewLLMClient(
		adapters.NewExternalLLMGateway(c.endpoint, c.virtualKey),
		opts.Provider,
		opts.Model,
	)
}
