package sdk

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services/project"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

type SDK struct {
	endpoint   string
	projectId  uuid.UUID
	virtualKey string
	directMode bool
	llmConfigs gateway.ConfigStore
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

func New(opts *ClientOptions) (*SDK, error) {
	// If endpoint is not provided, the SDK will operate in direct mode
	if opts.Endpoint == "" {
		// For direct mode, LLM config is necessary
		if opts.LLMConfigs == nil {
			return nil, fmt.Errorf("no LLM config store")
		}

		return &SDK{
			llmConfigs: opts.LLMConfigs,
			projectId:  uuid.New(),
			directMode: true,
		}, nil
	}

	// If project name is not provided, the SDK will operate as LLM gateway only mode
	if opts.ProjectName == "" {
		return &SDK{
			endpoint:   opts.Endpoint,
			virtualKey: opts.VirtualKey,
		}, nil
	}

	// If both endpoint and project name is provided, the SDK will operate as both LLM gateway, and agent server

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
			return &SDK{
				endpoint:   opts.Endpoint,
				projectId:  proj.ID,
				virtualKey: opts.VirtualKey,
			}, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", opts.ProjectName)
}

func (c *SDK) NewConversationManager(namespace, msgId, previousMsgId string, opts ...history.ConversationManagerOptions) core.ChatHistory {
	return history.NewConversationManager(
		c.getConversationPersistence(),
		c.projectId,
		namespace,
		msgId,
		previousMsgId,
		opts...,
	)
}

func (c *SDK) NewPromptManager(name string, label string, resolver core.SystemPromptResolver) core.SystemPromptProvider {
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
func (c *SDK) NewLLM(opts LLMOptions) llm.Provider {
	return gateway.NewLLMClient(
		c.getGatewayAdapter(),
		opts.Provider,
		opts.Model,
	)
}

func (c *SDK) getGatewayAdapter() gateway.LLMGatewayAdapter {
	if c.directMode {
		return adapters.NewLocalLLMGateway(gateway.NewLLMGateway(c.llmConfigs))
	}

	return adapters.NewExternalLLMGateway(c.endpoint, c.virtualKey)
}

func (c *SDK) getConversationPersistence() history.ConversationPersistenceManager {
	if c.directMode {
		return nil
	}

	return adapters.NewExternalConversationPersistence(c.endpoint)
}

func (c *SDK) NewAgent(options *agents.AgentOptions) *agents.Agent {
	return agents.NewAgent(options)
}
