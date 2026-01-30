package builder

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/sandbox"
)

type AgentBuilder struct {
	llmGateway     *gateway.LLMGateway
	svc            *services.Services
	broker         core.StreamBroker
	sandboxManager sandbox.Manager
}

func NewAgentBuilder(svc *services.Services, llmGateway *gateway.LLMGateway, broker core.StreamBroker, sandboxManager sandbox.Manager) *AgentBuilder {
	return &AgentBuilder{
		svc:            svc,
		llmGateway:     llmGateway,
		broker:         broker,
		sandboxManager: sandboxManager,
	}
}

func (b *AgentBuilder) BuildAndExecuteAgent(ctx context.Context, agentConfig *agent_config.AgentConfig, in *agents.AgentInput, key string) (*agents.AgentOutput, error) {
	projectID := agentConfig.ProjectID

	// Build prompt
	instruction := BuildPrompt(b.svc.Prompt, projectID, agentConfig.Config.Prompt, agentConfig.Config.Skills)

	// Model Configuration
	modelParams, err := BuildModelParams(agentConfig.Config.Model)
	if err != nil {
		return nil, err
	}

	// Structured Output
	var outputFormat map[string]any
	if agentConfig.Config.Schema != nil && agentConfig.Config.Schema.Name != "" {
		if err = sonic.Unmarshal(*agentConfig.Config.Schema.Schema, &outputFormat); err != nil {
			return nil, err
		}
	}

	// LLM Client
	llmClient := BuildLLMClient(
		b.llmGateway,
		key,
		llm.ProviderName(agentConfig.Config.Model.ProviderType),
		agentConfig.Config.Model.ModelID,
	)

	// History
	cm, err := BuildConversationManager(b.svc, projectID, b.llmGateway, agentConfig.Config.History, key)
	if err != nil {
		return nil, err
	}

	// MCP Servers
	var mcpProxies []agents.MCPToolset
	for _, mcpServerConfig := range agentConfig.Config.MCPServers {
		mcpClient, err := BuildMCPClient(&mcpServerConfig)
		if err != nil {
			return nil, err
		}
		mcpProxies = append(mcpProxies, mcpClient)
	}

	// Tools
	toolList := BuildToolsList(agentConfig.Config.Tools, b.sandboxManager)

	// Agent
	return agents.NewAgent(&agents.AgentOptions{
		Name:        agentConfig.GetName(),
		Instruction: instruction,
		Parameters:  modelParams,
		LLM:         llmClient,
		Output:      outputFormat,
		History:     cm,
		McpServers:  mcpProxies,
		Tools:       toolList,
		Runtime:     nil,
		MaxLoops:    agentConfig.Config.MaxIteration,
	}).Execute(ctx, in)
}
