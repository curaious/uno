package temporal_agent_builder

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm/responses"
	"go.temporal.io/sdk/workflow"
)

type AgentBuilder struct {
	llmGateway *gateway.LLMGateway
	svc        *services.Services
	broker     core.StreamBroker
}

func NewAgentBuilder(svc *services.Services, llmGateway *gateway.LLMGateway, broker core.StreamBroker) *AgentBuilder {
	return &AgentBuilder{
		svc:        svc,
		llmGateway: llmGateway,
		broker:     broker,
	}
}

func (b *AgentBuilder) BuildAndExecuteAgent(ctx workflow.Context, agentConfig *agent_config.AgentConfig, in *agents.AgentInput, key string) (*agents.AgentOutput, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	workflowId := workflow.GetInfo(ctx).WorkflowExecution.ID
	cb := func(chunk *responses.ResponseChunk) {
		b.broker.Publish(context.Background(), workflowId, chunk)
	}
	defer b.broker.Close(context.Background(), workflowId)

	// Project
	projectID := agentConfig.ProjectID

	// Build prompt
	instruction := NewTemporalPromptProxy(ctx, projectID, agentConfig.Config.Prompt)

	// Model Configuration
	modelParams, err := builder.BuildModelParams(agentConfig.Config.Model)
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
	llmClient := NewTemporalLLMProxy(ctx, agentConfig.Config.Model, key)

	// History
	var conversationManager *history.CommonConversationManager
	if agentConfig.Config.History != nil {
		conversationPersistenceProxy := NewTemporalConversationPersistenceProxy(ctx, projectID, agentConfig.Config.History)
		var options []history.ConversationManagerOptions
		if agentConfig.Config.History.Summarizer != nil && agentConfig.Config.History.Summarizer.Type != "none" {
			conversationSummarizerProxy := NewTemporalConversationSummarizerProxy(ctx, projectID, agentConfig.Config.History, key)
			options = append(options, history.WithSummarizer(conversationSummarizerProxy))
		}
		conversationManager = history.NewConversationManager(conversationPersistenceProxy, options...)
	}

	// MCP Servers
	var mcpProxies []agents.MCPToolset
	for _, mcpServerConfig := range agentConfig.Config.MCPServers {
		mcpProxy := NewTemporalMCPProxy(ctx, &mcpServerConfig)
		mcpProxies = append(mcpProxies, mcpProxy)
	}

	// Agent
	return agents.NewAgent(&agents.AgentOptions{
		Name:        agentConfig.Name,
		Instruction: instruction,
		Parameters:  modelParams,
		Output:      outputFormat,
		History:     conversationManager,
		McpServers:  mcpProxies,
		Tools:       nil,
		Runtime:     nil,
		MaxLoops:    agentConfig.Config.MaxIteration,
	}).WithLLM(llmClient).ExecuteWithExecutor(context.Background(), in, cb)
}
