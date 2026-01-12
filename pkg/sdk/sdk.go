package sdk

import (
	"fmt"
	"net/http"

	"github.com/curaious/uno/internal/services/project"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/sdk/adapters"
	"github.com/google/uuid"
)

type SDK struct {
	endpoint       string
	projectId      uuid.UUID
	virtualKey     string
	directMode     bool
	llmConfigs     gateway.ConfigStore
	restateConfig  RestateConfig
	temporalConfig TemporalConfig

	agents map[string]*agents.Agent
}

type ServerConfig struct {
	// Endpoint of the Uno Server
	Endpoint string

	// For LLM calls
	VirtualKey string

	// For conversations
	ProjectName string
}

type RestateConfig struct {
	Endpoint string
}

type TemporalConfig struct {
	Endpoint string
}

type ClientOptions struct {
	ServerConfig ServerConfig

	// Set this if you are using the SDK without the LLM Gateway server.
	// If `LLMConfigs` is set, then `ApiKey` will be ignored.
	LLMConfigs gateway.ConfigStore

	RestateConfig RestateConfig
}

func New(opts *ClientOptions) (*SDK, error) {
	if opts.LLMConfigs == nil && opts.ServerConfig.Endpoint == "" {
		return nil, fmt.Errorf("must provide either ServerConfig.Endpoint or LLMConfigs")
	}

	sdk := &SDK{
		llmConfigs:    opts.LLMConfigs,
		directMode:    opts.LLMConfigs != nil,
		endpoint:      opts.ServerConfig.Endpoint,
		virtualKey:    opts.ServerConfig.VirtualKey,
		restateConfig: opts.RestateConfig,
		agents:        map[string]*agents.Agent{},
	}

	if opts.ServerConfig.ProjectName == "" {
		return sdk, nil
	}

	// Convert project name to ID
	resp, err := http.DefaultClient.Get(fmt.Sprintf("%s/api/agent-server/projects", opts.ServerConfig.Endpoint))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	projectsRes := adapters.Response[[]project.Project]{}
	if err := utils.DecodeJSON(resp.Body, &projectsRes); err != nil {
		return nil, err
	}

	for _, proj := range projectsRes.Data {
		if proj.Name == opts.ServerConfig.ProjectName {
			sdk.projectId = proj.ID
			return sdk, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", opts.ServerConfig.ProjectName)
}
