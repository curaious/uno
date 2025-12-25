package service

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/praveen001/uno/internal/adapters"
	"github.com/praveen001/uno/internal/config"
	"github.com/praveen001/uno/internal/pubsub"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/internal/services/agent"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	restateExec "github.com/praveen001/uno/pkg/agent-framework/providers/restate"
	"github.com/praveen001/uno/pkg/agent-framework/summariser"
	"github.com/praveen001/uno/pkg/agent-framework/tools"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/redis/go-redis/v9"
	restate "github.com/restatedev/sdk-go"
	"go.opentelemetry.io/otel"
)

// ============================================================================
// Types - Same input format as /api/agent-server/converse
// ============================================================================

// AgentRunInput matches the ConverseRequest from converse.go
// UI sends same payload to Restate as it would to /api/agent-server/converse
type AgentRunInput struct {
	Message           responses.InputMessageUnion `json:"message"`
	Namespace         string                      `json:"namespace"`
	PreviousMessageID string                      `json:"previous_message_id,omitempty"`
	Context           map[string]any              `json:"context,omitempty"`
	SessionID         string                      `json:"session_id"`

	// These are passed via URL path/query in converse, but included in body for Restate
	ProjectID  string `json:"project_id"`
	AgentName  string `json:"agent_name"`
	VirtualKey string `json:"virtual_key,omitempty"` // Optional - falls back to project default
}

// AgentRunOutput is returned when the agent completes
type AgentRunOutput struct {
	FinalMessage []responses.OutputMessageUnion `json:"final_message"`
}

// AgentStatus for querying workflow state
type AgentStatus struct {
	Status        string `json:"status"`
	CurrentStep   string `json:"current_step"`
	LoopCount     int    `json:"loop_count"`
	CurrentTool   string `json:"current_tool,omitempty"`
	StreamChannel string `json:"stream_channel"`
	Error         string `json:"error,omitempty"`
}

// StreamEvent is published to Redis for UI streaming
type StreamEvent struct {
	RunID     string                   `json:"run_id"`
	Type      string                   `json:"type"`
	Message   *responses.ResponseChunk `json:"message,omitempty"`
	Status    string                   `json:"status,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Timestamp int64                    `json:"timestamp"`
}

// ============================================================================
// Global State
// ============================================================================

var (
	redisClient    *redis.Client
	svc            *services.Services
	llmGateway     *gateway.LLMGateway
	mcpServerCache = &sync.Map{} // session/serverID -> *tools.MCPServer
	tracer         = otel.Tracer("AgentWorkflow.Service")
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, skipping")
	}

	// Initialize services (uses same DB config as main server)
	conf := config.ReadConfig()
	svc = services.NewServices(conf)
	slog.Info("services initialized")

	// Create shared config store for the LLM gateway
	configStore := adapters.NewServiceConfigStore(svc.Provider, svc.VirtualKey)

	// Create pubsub for live configuration updates
	ps := pubsub.NewPubSub(conf)

	// Subscribe config store to pubsub before starting
	configStore.SubscribeToPubSub(ps)

	// Start pubsub listener
	if err := ps.Start(); err != nil {
		slog.Warn("Failed to start pubsub, config changes won't be live-reloaded", slog.Any("error", err))
	}

	// Initialize LLM gateway with shared config store
	llmGateway = gateway.NewLLMGateway(configStore)
	slog.Info("LLM gateway initialized with pubsub")

	slog.Info("config", slog.Any("config", conf))

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conf.REDIS_HOST, conf.REDIS_PORT),
		DB:       10,
		Username: conf.REDIS_USERNAME,
		Password: conf.REDIS_PASSWORD,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
}

// ============================================================================
// Agent Workflow
// ============================================================================

type RestateContext struct {
	restate.WorkflowContext
}

type AgentWorkflow struct{}

// Run executes the agent with durable checkpoints
// Fetches all config from DB like converse.go does
func (w AgentWorkflow) Run(reStateCtx restate.WorkflowContext, input AgentRunInput) (AgentRunOutput, error) {
	runID := restate.Key(reStateCtx)

	//h := http.Header{}
	//h.Add("traceparent", reStateCtx.Request().AttemptHeaders["Traceparent"][0])
	//carrier := propagation.HeaderCarrier(h)

	// Create a new span context with the custom trace ID
	//spanCtx := otel.GetTextMapPropagator().Extract(reStateCtx, carrier)
	//ctx := context.WithValue(spanCtx, "restateContent", reStateCtx)
	//ctx, span := tracer.Start(ctx, "AgentWorkflow.Run")
	//defer span.End()
	ctx := reStateCtx

	streamChannel := fmt.Sprintf("stream:%s", runID)

	slog.Info("agent workflow started", "run_id", runID, "project_id", input.ProjectID, "agent_name", input.AgentName)

	// Create RestateExecutor
	executor := restateExec.NewRestateExecutor(reStateCtx)

	publishStreamEvent(streamChannel, StreamEvent{
		RunID: runID, Type: "status", Status: "started", Timestamp: now(),
	})

	// Parse project ID
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return AgentRunOutput{}, fmt.Errorf("invalid project_id: %w", err)
	}

	// Fetch project to get default key
	_, projectDbSpan := tracer.Start(ctx, "DB.GetProjectByID")
	project, err := svc.Project.GetByID(ctx, projectID)
	if err != nil {
		projectDbSpan.RecordError(err)
		projectDbSpan.End()
		//span.End()
		publishStreamEvent(streamChannel, StreamEvent{
			RunID: runID, Type: "error", Error: "Failed to fetch project: " + err.Error(), Timestamp: now(),
		})
		return AgentRunOutput{}, fmt.Errorf("failed to fetch project: %w", err)
	}
	projectDbSpan.End()

	// Fetch agent configuration
	agentConfig, err := svc.Agent.GetByName(ctx, projectID, input.AgentName)
	if err != nil {
		publishStreamEvent(streamChannel, StreamEvent{
			RunID: runID, Type: "error", Error: "Failed to fetch agent: " + err.Error(), Timestamp: now(),
		})
		return AgentRunOutput{}, fmt.Errorf("failed to fetch agent: %w", err)
	}

	// Get virtual key
	virtualKey := input.VirtualKey
	if virtualKey == "" && project.DefaultKey != nil && *project.DefaultKey != "" {
		virtualKey = *project.DefaultKey
	}
	if virtualKey == "" {
		return AgentRunOutput{}, fmt.Errorf("virtual key is required")
	}

	// Build context data for template rendering
	contextData := map[string]any{
		"Env":     utils.EnvironmentVariables(),
		"Context": input.Context,
	}

	// Get provider type and model params
	providerType := llm.ProviderName(*agentConfig.ModelProviderType)
	var modelParams *responses.Parameters
	if agentConfig.ModelParameters != nil {
		if err := json.Unmarshal(*agentConfig.ModelParameters, &modelParams); err != nil {
			return AgentRunOutput{}, err
		}
	}

	// Build LLM client
	llmClient := gateway.NewLLMClient(
		adapters.NewInternalLLMGateway(llmGateway, virtualKey),
		providerType,
		agentConfig.ModelName,
	)

	// Fetch and connect to MCP servers
	allTools, err := fetchAndConnectMCPServers(ctx, projectID, agentConfig, input.SessionID, contextData)
	if err != nil {
		slog.Warn("some MCP servers failed", "error", err)
	}

	// Get prompt label
	promptLabel := "latest"
	if agentConfig.PromptLabel != nil && *agentConfig.PromptLabel != "" {
		promptLabel = *agentConfig.PromptLabel
	}

	// Create instruction provider
	instructionProvider := prompts.NewWithLoader(
		adapters.NewInternalPromptPersistence(svc.Prompt, projectID, agentConfig.PromptName, promptLabel),
		prompts.WithDefaultResolver(contextData),
	)

	// Build conversation manager options
	conversationManagerOpts := []history.ConversationManagerOptions{}

	// Setup summarizer if enabled
	var summarizer core.HistorySummarizer
	if agentConfig.EnableHistory && agentConfig.SummarizerType != nil && *agentConfig.SummarizerType != "none" {
		summarizer, err = buildSummarizer(agentConfig, projectID, virtualKey, contextData)
		if err != nil {
			slog.Warn("failed to build summarizer", "error", err)
		}
		if summarizer != nil {
			conversationManagerOpts = append(conversationManagerOpts, history.WithSummarizer(summarizer))
		}
	}

	slog.Info("agent configured",
		"run_id", runID,
		"provider", providerType,
		"model", agentConfig.ModelName,
		"tools_count", len(allTools),
		"history_enabled", agentConfig.EnableHistory,
	)

	// Create DurableAgent
	agentOpts := &agents.DurableAgentOptions{
		Name:                agentConfig.Name,
		Executor:            executor,
		MaxLoops:            50,
		LLM:                 llmClient,
		Tools:               allTools,
		InstructionProvider: instructionProvider,
	}

	// If agent has a schema configured, set it as the output format
	if agentConfig.SchemaData != nil {
		var schemaMap map[string]any
		if err := json.Unmarshal(*agentConfig.SchemaData, &schemaMap); err == nil {
			agentOpts.Output = schemaMap
		}
	}

	// Add history manager if enabled
	if agentConfig.EnableHistory {
		agentOpts.History = history.NewConversationManager(
			adapters.NewInternalConversationPersistence(svc.Conversation, projectID),
			input.Namespace,
			input.PreviousMessageID,
			conversationManagerOpts...,
		)
	}

	agent, err := agents.NewDurableAgent(agentOpts)
	if err != nil {
		publishStreamEvent(streamChannel, StreamEvent{
			RunID: runID, Type: "error", Error: err.Error(), Timestamp: now(),
		})
		return AgentRunOutput{}, fmt.Errorf("failed to create agent: %w", err)
	}

	// Streaming callback
	streamCallback := func(chunk *responses.ResponseChunk) {
		eventType := "message"
		publishStreamEvent(streamChannel, StreamEvent{
			RunID: runID, Type: eventType, Message: chunk, Timestamp: now(),
		})
	}

	// Execute the agent
	result, err := agent.Execute(ctx, []responses.InputMessageUnion{input.Message}, streamCallback)

	if err != nil {
		publishStreamEvent(streamChannel, StreamEvent{
			RunID: runID, Type: "error", Error: err.Error(), Timestamp: now(),
		})
		return AgentRunOutput{}, err
	}

	return AgentRunOutput{
		FinalMessage: result.Output,
	}, nil
}

// GetStatus returns the current workflow state
func (AgentWorkflow) GetStatus(ctx restate.WorkflowSharedContext) (AgentStatus, error) {
	status, _ := restate.Get[string](ctx, "status")
	currentStep, _ := restate.Get[string](ctx, "current_step")
	loopCount, _ := restate.Get[int](ctx, "loop_count")
	currentTool, _ := restate.Get[string](ctx, "current_tool")
	streamChannel, _ := restate.Get[string](ctx, "stream_channel")
	errorMsg, _ := restate.Get[string](ctx, "error")

	return AgentStatus{
		Status:        status,
		CurrentStep:   currentStep,
		LoopCount:     loopCount,
		CurrentTool:   currentTool,
		StreamChannel: streamChannel,
		Error:         errorMsg,
	}, nil
}

// Cancel signals the workflow to stop
func (AgentWorkflow) Cancel(ctx restate.WorkflowSharedContext, reason string) error {
	runID := restate.Key(ctx)
	slog.Info("cancel requested", "run_id", runID, "reason", reason)

	if redisClient != nil {
		cancelKey := fmt.Sprintf("cancel:%s", runID)
		redisClient.Set(ctx, cancelKey, reason, time.Hour)
		publishStreamEvent(fmt.Sprintf("stream:%s", runID), StreamEvent{
			RunID: runID, Type: "status", Status: "cancel_requested", Timestamp: now(),
		})
	}
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func fetchAndConnectMCPServers(ctx context.Context, projectID uuid.UUID, agentConfig *agent.AgentWithDetails, sessionID string, contextData map[string]any) ([]core.Tool, error) {
	if len(agentConfig.MCPServers) == 0 {
		return nil, nil
	}

	mcpServerIDsToFetch := []uuid.UUID{}
	mcpServerCacheMap := make(map[uuid.UUID]*tools.MCPServer)

	// Check cache first
	for _, agentMCP := range agentConfig.MCPServers {
		cacheKey := fmt.Sprintf("%s/%s", sessionID, agentMCP.MCPServerID)
		if cached, ok := mcpServerCache.Load(cacheKey); ok {
			mcpServer := cached.(*tools.MCPServer)
			if err := mcpServer.Client.Ping(ctx); err == nil {
				mcpServerCacheMap[agentMCP.MCPServerID] = mcpServer
				continue
			}
			mcpServerCache.Delete(cacheKey)
		}
		mcpServerIDsToFetch = append(mcpServerIDsToFetch, agentMCP.MCPServerID)
	}

	// Fetch uncached servers from DB
	if len(mcpServerIDsToFetch) > 0 {
		mcpServerConfigs, err := svc.MCPServer.GetByIDs(ctx, projectID, mcpServerIDsToFetch)
		if err != nil {
			slog.Warn("Failed to fetch MCP servers", "error", err)
		} else {
			for _, mcpServerID := range mcpServerIDsToFetch {
				mcpServerConfig, exists := mcpServerConfigs[mcpServerID]
				if !exists {
					continue
				}

				// Process headers with template
				headers := make(map[string]string)
				if mcpServerConfig.Headers != nil {
					for k, v := range mcpServerConfig.Headers {
						headers[k] = utils.TryAndParseAsTemplate(v, contextData)
					}
				}

				mcpServer, err := tools.NewMCPServer(ctx, mcpServerConfig.Endpoint, headers)
				if err != nil {
					slog.Warn("Failed to connect to MCP server",
						"server_id", mcpServerID.String(),
						"endpoint", mcpServerConfig.Endpoint,
						"error", err,
					)
					continue
				}

				cacheKey := fmt.Sprintf("%s/%s", sessionID, mcpServerID)
				mcpServerCache.Store(cacheKey, mcpServer)
				mcpServerCacheMap[mcpServerID] = mcpServer
			}
		}
	}

	// Collect tools from all servers
	var allTools []core.Tool
	for _, agentMCP := range agentConfig.MCPServers {
		mcpServer, exists := mcpServerCacheMap[agentMCP.MCPServerID]
		if !exists {
			continue
		}

		var toolFilters []string
		if len(agentMCP.ToolFilters) > 0 {
			toolFilters = agentMCP.ToolFilters
		}

		mcpTools := mcpServer.GetTools(tools.WithMcpToolFilter(toolFilters...))
		allTools = append(allTools, mcpTools...)
	}

	return allTools, nil
}

func buildSummarizer(agentConfig *agent.AgentWithDetails, projectID uuid.UUID, virtualKey string, contextData map[string]any) (core.HistorySummarizer, error) {
	switch *agentConfig.SummarizerType {
	case "llm":
		if agentConfig.SummarizerModelModelID == nil || agentConfig.SummarizerProviderType == nil {
			return nil, fmt.Errorf("summarizer model configuration incomplete")
		}
		if agentConfig.SummarizerPromptName == nil || *agentConfig.SummarizerPromptName == "" {
			return nil, fmt.Errorf("summarizer prompt name missing")
		}

		summarizerPromptLabel := "latest"
		if agentConfig.LLMSummarizerPromptLabel != nil && *agentConfig.LLMSummarizerPromptLabel != "" {
			summarizerPromptLabel = *agentConfig.LLMSummarizerPromptLabel
		}

		summarizerInstructionProvider := prompts.NewWithLoader(
			adapters.NewInternalPromptPersistence(svc.Prompt, projectID, *agentConfig.SummarizerPromptName, summarizerPromptLabel),
			prompts.WithDefaultResolver(contextData),
		)

		tokenThreshold := 500
		if agentConfig.LLMSummarizerTokenThreshold != nil && *agentConfig.LLMSummarizerTokenThreshold > 0 {
			tokenThreshold = *agentConfig.LLMSummarizerTokenThreshold
		}

		keepRecentCount := 5
		if agentConfig.LLMSummarizerKeepRecentCount != nil && *agentConfig.LLMSummarizerKeepRecentCount >= 0 {
			keepRecentCount = *agentConfig.LLMSummarizerKeepRecentCount
		}

		summarizerProviderType := llm.ProviderName(*agentConfig.SummarizerProviderType)
		var summarizerModelParams responses.Parameters
		if agentConfig.SummarizerModelParameters != nil {
			json.Unmarshal(*agentConfig.SummarizerModelParameters, &summarizerModelParams)
		}

		summarizerLLM := gateway.NewLLMClient(
			adapters.NewInternalLLMGateway(llmGateway, virtualKey),
			summarizerProviderType,
			*agentConfig.SummarizerModelModelID,
		)

		return summariser.NewLLMHistorySummarizer(&summariser.LLMHistorySummarizerOptions{
			LLM:                 summarizerLLM,
			InstructionProvider: summarizerInstructionProvider,
			TokenThreshold:      tokenThreshold,
			KeepRecentCount:     keepRecentCount,
		}), nil

	case "sliding_window":
		if agentConfig.SlidingWindowKeepCount == nil || *agentConfig.SlidingWindowKeepCount <= 0 {
			return nil, fmt.Errorf("sliding_window_keep_count is required")
		}
		return summariser.NewSlidingWindowHistorySummarizer(&summariser.SlidingWindowHistorySummarizerOptions{
			KeepCount: *agentConfig.SlidingWindowKeepCount,
		}), nil
	}

	return nil, nil
}

func publishStreamEvent(channel string, event StreamEvent) {
	if redisClient == nil {
		return
	}

	data, err := json.Marshal(event.Message)
	if err != nil {
		return
	}

	redisClient.Publish(context.Background(), channel, data)
}

func now() int64 {
	return time.Now().UnixMilli()
}
