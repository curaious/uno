package sdk

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services/project"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway"
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
