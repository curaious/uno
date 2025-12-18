package sdk

import (
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/gateway/client"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

type Client struct {
	endpoint   string
	virtualKey string
	llmConfigs gateway.ConfigStore

	directLLMCalls bool
}

type ClientOptions struct {
	// Endpoint of the LLM Gateway server.
	Endpoint string

	// Set with the virtual key obtained from the LLM gateway server.
	VirtualKey string

	// Set this if you are using the SDK without the LLM Gateway server.
	// If `LLMConfigs` is set, then `ApiKey` will be ignored.
	LLMConfigs gateway.ConfigStore
}

func NewClient(opts *ClientOptions) (*Client, error) {
	if opts.LLMConfigs != nil {
		return &Client{
			llmConfigs:     opts.LLMConfigs,
			directLLMCalls: true,
		}, nil
	}

	return &Client{
		endpoint:   opts.Endpoint,
		virtualKey: opts.VirtualKey,
	}, nil
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
