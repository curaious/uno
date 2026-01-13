package controllers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/adapters"
	"github.com/curaious/uno/internal/api/response"
	"github.com/curaious/uno/internal/perrors"
	"github.com/curaious/uno/internal/services"
	agent2 "github.com/curaious/uno/internal/services/agent"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
	"github.com/curaious/uno/pkg/agent-framework/prompts"
	"github.com/curaious/uno/pkg/agent-framework/summariser"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer     = otel.Tracer("Controller")
	agentCache = utils.NewTTLSyncMap(1*time.Minute, 5*time.Second)
)

type ConverseRequest struct {
	Message           responses.InputMessageUnion `json:"message" doc:"User message"`
	Namespace         string                      `json:"namespace" doc:"Namespace ID"`
	PreviousMessageID string                      `json:"previous_message_id" doc:"Previous run ID for threading"`
	Context           map[string]any              `json:"context" doc:"Context to pass to prompt template"`
	SessionID         string                      `json:"session_id" required:"true" doc:"Session ID"`
}

func RegisterConverseRoute(r *router.Router, svc *services.Services, llmGateway *gateway.LLMGateway) {
	r.POST("/api/agent-server/converse", func(reqCtx *fasthttp.RequestCtx) {
		ctx, span := tracer.Start(requestContext(reqCtx), "Controller.Converse")
		reqCtx.Response.Header.Set("X-Trace-Id", span.SpanContext().TraceID().String())

		projectIDStr := string(reqCtx.QueryArgs().Peek("project_id"))
		if projectIDStr == "" {
			writeError(reqCtx, ctx, "Project ID is required", perrors.NewErrInvalidRequest("project_id parameter is required", errors.New("project_id parameter is required")))
			span.End()
			return
		}
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			writeError(reqCtx, ctx, "Invalid project ID", perrors.NewErrInvalidRequest("project_id is invalid", errors.New("project_id is invalid")))
			span.End()
			return
		}

		agentName := string(reqCtx.QueryArgs().Peek("agent_name"))
		if agentName == "" {
			writeError(reqCtx, ctx, "Agent name is required", perrors.NewErrInvalidRequest("agent_name parameter is required", errors.New("agent_name parameter is required")))
			span.End()
			return
		}

		var agent *agents.Agent
		agentCacheKey := fmt.Sprintf("agent-%s-%s", agentName, projectID.String())

		agentAny, exists := agentCache.Get(agentCacheKey)
		if !exists {
			agent, err = buildAgent(ctx, svc, llmGateway, projectID, agentName)
			if err != nil {
				writeError(reqCtx, ctx, "Error occurred while building the agent", err)
				span.End()
				return
			}
			agentCache.Set(agentCacheKey, agent)
		} else {
			agent = agentAny.(*agents.Agent)
		}

		// Parse request body first to get message_id for trace ID
		var reqPayload ConverseRequest
		if err := json.Unmarshal(reqCtx.PostBody(), &reqPayload); err != nil {
			writeError(reqCtx, ctx, "Invalid request body", perrors.NewErrInternalServerError(err.Error(), err))
			span.End()
			return
		}

		span.SetAttributes(
			attribute.String("project_id", projectIDStr),
			attribute.String("agent_name", agentName),
			attribute.String("namespace", reqPayload.Namespace),
			attribute.String("session_id", reqPayload.SessionID),
		)

		reqCtx.Response.Header.Set("Content-Type", "text/event-stream")
		reqCtx.Response.Header.Set("Cache-Control", "no-cache")
		reqCtx.SetStatusCode(fasthttp.StatusOK)

		reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
			// End the controller span when streaming completes
			defer span.End()
			defer w.Flush()

			reqHeaders := map[string]string{}
			reqCtx.Request.Header.VisitAll(func(key, value []byte) {
				reqHeaders[strings.ReplaceAll(string(key), "-", "_")] = string(value)
			})

			contextData := map[string]any{
				"Env":     utils.EnvironmentVariables(),
				"Context": reqPayload.Context,
				"Header":  reqHeaders,
			}

			_, execErr := agent.Execute(
				ctx,
				&agents.AgentInput{
					Namespace:         reqPayload.Namespace,
					PreviousMessageID: reqPayload.PreviousMessageID,
					Messages:          []responses.InputMessageUnion{reqPayload.Message},
					RunContext:        contextData,
					Callback: func(message *responses.ResponseChunk) {
						buf, err := json.Marshal(message)
						if err != nil {
							return
						}

						_, _ = fmt.Fprintf(w, "event: %s\n", message.ChunkType())
						_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)
						_ = w.Flush()
					},
				},
			)
			if execErr != nil {
				span.RecordError(execErr)
				span.SetStatus(codes.Error, execErr.Error())
				errResp := response.NewResponse[any](ctx, "agent execution failed", nil).
					WithError(perrors.NewErrInternalServerError(execErr.Error(), execErr))
				buf, err := json.Marshal(errResp)
				if err == nil {
					_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)
					_ = w.Flush()
				}
			}
		})
	})
}

func buildAgent(ctx context.Context, svc *services.Services, llmGateway *gateway.LLMGateway, projectID uuid.UUID, agentName string) (*agents.Agent, error) {
	span := trace.SpanFromContext(ctx)

	_, projectDbSpan := tracer.Start(ctx, "DB.GetProjectByID")
	project, err := svc.Project.GetByID(ctx, projectID)
	if err != nil {
		projectDbSpan.RecordError(err)
		projectDbSpan.End()
		return nil, perrors.NewErrInternalServerError("Failed to fetch project", errors.New("Failed to fetch project: "+err.Error()))
	}
	projectDbSpan.End()

	// Fetch agent configuration (includes model information)
	_, dbSpan := tracer.Start(ctx, "DB.GetAgentByName")
	agentConfig, err := svc.Agent.GetByName(ctx, projectID, agentName)
	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.End()
		return nil, perrors.NewErrInternalServerError("Failed to fetch agent", errors.New("Failed to fetch agent: "+err.Error()))
	}
	dbSpan.End()

	// Get default API key for the provider type
	providerType := llm.ProviderName(*agentConfig.ModelProviderType)

	// Get virtual key: prefer project's default key, fallback to header
	virtualKey := ""
	if project.DefaultKey != nil && *project.DefaultKey != "" {
		virtualKey = *project.DefaultKey
	}

	if virtualKey == "" {
		return nil, perrors.NewErrInvalidRequest("Virtual key is required", errors.New("virtual key is required"))
	}

	// Parse model parameters from model configuration - directly unmarshal into ModelParameters
	var modelParams *responses.Parameters
	if agentConfig.ModelParameters != nil {
		if err := json.Unmarshal(*agentConfig.ModelParameters, &modelParams); err != nil {
			return nil, perrors.NewErrInternalServerError("unable to parse model parameters", errors.New("Model parameter is invalid: "+err.Error()))
		}
	}

	if modelParams.Reasoning != nil {
		modelParams.Include = []responses.Includable{responses.IncludableReasoningEncryptedContent}
		modelParams.Reasoning.Summary = utils.Ptr("auto")
	}

	// Build LLM client using internal provider (server-side with virtual key resolution)
	llmClient := gateway.NewLLMClient(
		adapters.NewInternalLLMGateway(llmGateway, virtualKey),
		providerType,
		agentConfig.ModelName,
	)

	// Fetch and connect to MCP servers
	mcpServerIDsToFetch := []uuid.UUID{}
	agentMcpConfigs := map[uuid.UUID]*agent2.AgentMCPServerDetail{}
	for _, agentMCP := range agentConfig.MCPServers {
		agentMcpConfigs[agentMCP.MCPServerID] = agentMCP
		mcpServerIDsToFetch = append(mcpServerIDsToFetch, agentMCP.MCPServerID)
	}

	mcpServers := make([]agents.MCPToolset, len(mcpServerIDsToFetch))
	if len(mcpServerIDsToFetch) > 0 {
		// Trace MCP server config fetch from DB
		_, mcpDbSpan := tracer.Start(ctx, "DB.GetMCPServersByIDs")
		mcpDbSpan.SetAttributes(attribute.Int("mcp.count", len(mcpServerIDsToFetch)))
		mcpServerConfigs, err := svc.MCPServer.GetByIDs(ctx, projectID, mcpServerIDsToFetch)
		if err != nil {
			mcpDbSpan.RecordError(err)
			slog.WarnContext(ctx, "Failed to batch fetch MCP servers", slog.Any("error", err))
		}
		mcpDbSpan.End()

		if err == nil {
			for idx, mcpServerID := range mcpServerIDsToFetch {
				mcpServerConfig, exists := mcpServerConfigs[mcpServerID]
				if !exists {
					continue
				}

				options := []mcpclient.McpServerOption{}
				if mcpServerConfig.Headers != nil && len(mcpServerConfig.Headers) > 0 {
					options = append(options, mcpclient.WithHeaders(mcpServerConfig.Headers))
				}

				agentMcpConfig, ok := agentMcpConfigs[mcpServerID]
				if ok {
					if agentMcpConfig.ToolFilters != nil && len(agentMcpConfig.ToolFilters) > 0 {
						options = append(options, mcpclient.WithToolFilter(agentMcpConfig.ToolFilters...))
					}

					if agentMcpConfig.ToolsRequiringHumanApproval != nil && len(agentMcpConfig.ToolsRequiringHumanApproval) > 0 {
						options = append(options, mcpclient.WithApprovalRequiredTools(agentMcpConfig.ToolsRequiringHumanApproval...))
					}
				}

				mcpServer, err := mcpclient.NewSSEClient(context.Background(), mcpServerConfig.Endpoint, options...)
				if err != nil {
					slog.WarnContext(ctx, "Failed to connect to MCP server", slog.String("server_id", mcpServerID.String()), slog.Any("error", err))
					continue
				}
				mcpServers[idx] = mcpServer
			}
		}
	}

	allTools := []core.Tool{}

	promptLabel := "latest"
	if agentConfig.PromptLabel != nil && *agentConfig.PromptLabel != "" {
		promptLabel = *agentConfig.PromptLabel
	}

	instructionProvider := prompts.NewWithLoader(
		adapters.NewInternalPromptPersistence(svc.Prompt, projectID, agentConfig.PromptName, promptLabel),
	)

	conversationManagerOpts := []history.ConversationManagerOptions{}

	var summarizer core.HistorySummarizer
	if agentConfig.EnableHistory && agentConfig.SummarizerType != nil && *agentConfig.SummarizerType != "none" {
		switch *agentConfig.SummarizerType {
		case "llm":
			if agentConfig.LLMSummarizerModelID == nil {
				return nil, perrors.NewErrInvalidRequest("LLM summarizer model ID is required", errors.New("llm_summarizer_model_id is required"))
			}
			if agentConfig.LLMSummarizerPromptID == nil {
				return nil, perrors.NewErrInvalidRequest("LLM summarizer prompt ID is required", errors.New("llm_summarizer_prompt_id is required"))
			}

			if agentConfig.SummarizerPromptName == nil || *agentConfig.SummarizerPromptName == "" {
				return nil, perrors.NewErrInvalidRequest("Summarizer prompt name is missing", errors.New("summarizer prompt name missing"))
			}

			summarizerPromptLabel := "latest"
			if agentConfig.LLMSummarizerPromptLabel != nil && *agentConfig.LLMSummarizerPromptLabel != "" {
				summarizerPromptLabel = *agentConfig.LLMSummarizerPromptLabel
			}

			summarizerInstructionProvider := prompts.NewWithLoader(
				adapters.NewInternalPromptPersistence(svc.Prompt, projectID, *agentConfig.SummarizerPromptName, summarizerPromptLabel),
			)

			tokenThreshold := 500
			if agentConfig.LLMSummarizerTokenThreshold != nil && *agentConfig.LLMSummarizerTokenThreshold > 0 {
				tokenThreshold = *agentConfig.LLMSummarizerTokenThreshold
			}

			keepRecentCount := 5
			if agentConfig.LLMSummarizerKeepRecentCount != nil && *agentConfig.LLMSummarizerKeepRecentCount >= 0 {
				keepRecentCount = *agentConfig.LLMSummarizerKeepRecentCount
			}

			if agentConfig.SummarizerModelModelID == nil || agentConfig.SummarizerProviderType == nil {
				return nil, perrors.NewErrInternalServerError("Summarizer model or provider information is missing", errors.New("summarizer configuration incomplete"))
			}

			// Build summarizer LLM using gateway with virtual key
			summarizerProviderType := llm.ProviderName(*agentConfig.SummarizerProviderType)
			var summarizerModelParams responses.Parameters
			if agentConfig.SummarizerModelParameters != nil {
				var params responses.Parameters
				if err := json.Unmarshal(*agentConfig.SummarizerModelParameters, &params); err == nil {
					summarizerModelParams = params
				}
			}
			summarizerLLM := gateway.NewLLMClient(
				adapters.NewInternalLLMGateway(llmGateway, virtualKey),
				summarizerProviderType,
				*agentConfig.SummarizerModelModelID,
			)

			summarizer = summariser.NewLLMHistorySummarizer(&summariser.LLMHistorySummarizerOptions{
				LLM:             summarizerLLM,
				Instruction:     summarizerInstructionProvider,
				TokenThreshold:  tokenThreshold,
				KeepRecentCount: keepRecentCount,
				Parameters:      summarizerModelParams,
			})

		case "sliding_window":
			if agentConfig.SlidingWindowKeepCount == nil || *agentConfig.SlidingWindowKeepCount <= 0 {
				return nil, perrors.NewErrInvalidRequest("sliding_window_keep_count is required and must be greater than 0 when summarizer_type is 'sliding_window'", errors.New("sliding_window_keep_count is required"))
			}

			summarizer = summariser.NewSlidingWindowHistorySummarizer(&summariser.SlidingWindowHistorySummarizerOptions{
				KeepCount: *agentConfig.SlidingWindowKeepCount,
			})
		}

		if summarizer != nil {
			conversationManagerOpts = append(conversationManagerOpts, history.WithSummarizer(summarizer))
		}
	}

	// Add trace attributes for agent configuration
	span.SetAttributes(
		attribute.String("llm.provider", string(providerType)),
		attribute.String("llm.model", agentConfig.ModelName),
		attribute.Int("tools.count", len(allTools)),
		attribute.Bool("history.enabled", agentConfig.EnableHistory),
	)

	agentOpts := &agents.AgentOptions{
		Name:        agentConfig.Name,
		LLM:         llmClient,
		Tools:       allTools,
		McpServers:  mcpServers,
		Instruction: instructionProvider,
		Parameters:  *modelParams,
	}

	// If agent has a schema configured, set it as the output format
	if agentConfig.SchemaData != nil {
		var schemaMap map[string]any
		if err := json.Unmarshal(*agentConfig.SchemaData, &schemaMap); err == nil {
			agentOpts.Output = schemaMap
		}
	}

	if agentConfig.EnableHistory {
		agentOpts.History = history.NewConversationManager(
			adapters.NewInternalConversationPersistence(svc.Conversation, projectID),
			conversationManagerOpts...,
		)
	}

	agent := agents.NewAgent(agentOpts)

	return agent, nil
}
