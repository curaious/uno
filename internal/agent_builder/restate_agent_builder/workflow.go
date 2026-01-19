package restate_agent_builder

import (
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/curaious/uno/pkg/sdk/runtime/restate_runtime"
	restate "github.com/restatedev/sdk-go"
	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("RestateWorkflow")
)

type WorkflowInput struct {
	AgentConfig *agent_config.AgentConfig
	Input       *agents.AgentInput
	Key         string
}

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

func (b *AgentBuilder) BuildAndExecuteAgent(ctx restate.WorkflowContext, in *WorkflowInput) (*agents.AgentOutput, error) {
	traceCtx, span := tracer.Start(ctx, "Restate.BuildAndExecuteAgent")
	defer span.End()

	ctx = restate.WrapContext(ctx, traceCtx)

	workflowId := restate.Key(ctx)
	cb := func(chunk *responses.ResponseChunk) {
		if err := b.broker.Publish(ctx, workflowId, chunk); err != nil {
			slog.WarnContext(ctx, "unable to publish chunk to broker", slog.Any("error", err))
		}
	}
	defer b.broker.Close(ctx, workflowId)

	// Project
	projectID := in.AgentConfig.ProjectID

	// Build prompt
	instruction := restate_runtime.NewRestatePrompt(ctx, builder.BuildPrompt(b.svc.Prompt, projectID, in.AgentConfig.Config.Prompt))

	// Model Configuration
	modelParams, err := builder.BuildModelParams(in.AgentConfig.Config.Model)
	if err != nil {
		return nil, err
	}

	// Structured Output
	var outputFormat map[string]any
	if in.AgentConfig.Config.Schema != nil && in.AgentConfig.Config.Schema.Name != "" {
		if err = sonic.Unmarshal(*in.AgentConfig.Config.Schema.Schema, &outputFormat); err != nil {
			return nil, err
		}
	}

	// LLM Client
	llmClient := restate_runtime.NewRestateLLM(
		ctx,
		builder.BuildLLMClient(
			b.llmGateway,
			in.Key,
			llm.ProviderName(in.AgentConfig.Config.Model.ProviderType),
			in.AgentConfig.Config.Model.ModelID,
		),
	)

	// History
	cm, err := builder.BuildConversationManager(b.svc, projectID, b.llmGateway, in.AgentConfig.Config.History, in.Key)
	if err != nil {
		return nil, err
	}
	var conversationManager *history.CommonConversationManager
	if in.AgentConfig.Config.History != nil {
		conversationPersistenceProxy := restate_runtime.NewRestateConversationPersistence(ctx, cm.ConversationPersistenceAdapter)
		var options []history.ConversationManagerOptions
		if in.AgentConfig.Config.History.Summarizer != nil && in.AgentConfig.Config.History.Summarizer.Type != "none" {
			conversationSummarizerProxy := restate_runtime.NewRestateConversationSummarizer(ctx, cm.Summarizer)
			options = append(options, history.WithSummarizer(conversationSummarizerProxy))
		}
		conversationManager = history.NewConversationManager(conversationPersistenceProxy, options...)
	}

	// MCP Servers
	var mcpProxies []agents.MCPToolset
	for _, mcpServerConfig := range in.AgentConfig.Config.MCPServers {
		mcpClient, err := builder.BuildMCPClient(&mcpServerConfig)
		if err != nil {
			return nil, err
		}
		mcpProxy := restate_runtime.NewRestateMCPServer(ctx, mcpClient)
		mcpProxies = append(mcpProxies, mcpProxy)
	}

	// Agent
	return agents.NewAgent(&agents.AgentOptions{
		Name:        in.AgentConfig.Name,
		Instruction: instruction,
		Parameters:  modelParams,
		Output:      outputFormat,
		History:     conversationManager,
		McpServers:  mcpProxies,
		Tools:       nil,
		Runtime:     nil,
	}).WithLLM(llmClient).ExecuteWithExecutor(ctx, in.Input, cb)
}
