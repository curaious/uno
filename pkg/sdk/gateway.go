package sdk

import (
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

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
