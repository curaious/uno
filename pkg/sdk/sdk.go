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
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/gateway/client"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

type Client struct {
	endpoint       string
	projectId      uuid.UUID
	virtualKey     string
	directLLMCalls bool
	llmConfigs     gateway.ConfigStore
}

type ClientOptions struct {
	// Endpoint of the LLM Gateway server.
	Endpoint string

	// Set with the virtual key obtained from the LLM gateway server.
	VirtualKey string

	// Set this if you are using the SDK without the LLM Gateway server.
	// If `LLMConfigs` is set, then `ApiKey` will be ignored.
	LLMConfigs gateway.ConfigStore

	ProjectName string
}

func NewClient(opts *ClientOptions) (*Client, error) {
	if opts.LLMConfigs != nil {
		return &Client{
			llmConfigs:     opts.LLMConfigs,
			projectId:      uuid.New(),
			directLLMCalls: true,
		}, nil
	}

	// Convert project name to ID
	url := fmt.Sprintf("%s/api/agent-server/projects", opts.Endpoint)

	// Load the list of projects from the LLM gateway
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

// NewLLM creates a new LLMClient that provides access to multiple LLM providers.
func (c *Client) NewLLM(opts LLMOptions) llm.Provider {
	return client.NewLLMClient(
		c.getGatewayAdapter(),
		opts.Provider,
		opts.Model,
	)
}

func (c *Client) getGatewayAdapter() client.LLMGateway {
	if c.directLLMCalls {
		return adapters.NewLocalLLMGateway(gateway.NewLLMGateway(c.llmConfigs))
	}

	return adapters.NewExternalLLMGateway(c.endpoint, c.virtualKey)
}
