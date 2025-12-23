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
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/adapters"
	"github.com/praveen001/uno/internal/api/response"
	"github.com/praveen001/uno/internal/perrors"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/agent-framework/prompts"
	"github.com/praveen001/uno/pkg/agent-framework/summariser"
	"github.com/praveen001/uno/pkg/agent-framework/tools"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer               = otel.Tracer("Controller")
	mcpServerCache       = utils.NewTTLSyncMap(10*time.Minute, 1*time.Minute)
	projectNameToIdCache = utils.NewTTLSyncMap(10*time.Minute, 1*time.Minute)
)

type ConverseRequest struct {
	Message           responses.InputMessageUnion `json:"message" doc:"User message"`
	Namespace         string                      `json:"namespace" doc:"Namespace ID"`
	MessageID         string                      `json:"message_id" required:"true" doc:"MessageID"`
	PreviousMessageID string                      `json:"previous_message_id" doc:"Previous run ID for threading"`
	Context           map[string]any              `json:"context" doc:"Context to pass to prompt template"`
	SessionID         string                      `json:"session_id" required:"true" doc:"Session ID"`
}

func RegisterConverseRoute(r *router.Router, svc *services.Services) {
	llmGateway := gateway.NewLLMGateway(adapters.NewServiceConfigStore(svc.Provider, svc.VirtualKey))

	r.POST("/api/agent-server/converse", func(reqCtx *fasthttp.RequestCtx) {
		baseCtx := requestContext(reqCtx)

		projectIDStr := string(reqCtx.QueryArgs().Peek("project_id"))
		if projectIDStr == "" {
			writeError(reqCtx, baseCtx, "Project ID is required", perrors.NewErrInvalidRequest("project_id parameter is required", errors.New("project_id parameter is required")))
			return
		}
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			writeError(reqCtx, baseCtx, "Invalid project ID", perrors.NewErrInvalidRequest("project_id is invalid", errors.New("project_id is invalid")))
			return
		}

		agentName := string(reqCtx.QueryArgs().Peek("agent_name"))
		if agentName == "" {
			writeError(reqCtx, baseCtx, "Agent name is required", perrors.NewErrInvalidRequest("agent_name parameter is required", errors.New("agent_name parameter is required")))
			return
		}

		// Parse request body first to get message_id for trace ID
		var reqPayload ConverseRequest
		if err := json.Unmarshal(reqCtx.PostBody(), &reqPayload); err != nil {
			writeError(reqCtx, baseCtx, "Invalid request body", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		// Create trace ID from message_id (UUID without dashes = 32 hex chars = 16 bytes)
		// This allows direct lookup of traces by message_id
		var ctx context.Context
		var span trace.Span
		if messageUUID, err := uuid.Parse(reqPayload.MessageID); err == nil {
			// Convert UUID bytes to trace ID (UUID is 16 bytes, same as trace ID)
			var traceIDBytes [16]byte
			copy(traceIDBytes[:], messageUUID[:])
			traceID := trace.TraceID(traceIDBytes)

			// Create a new span context with the custom trace ID
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				TraceFlags: trace.FlagsSampled,
			})
			ctx = trace.ContextWithSpanContext(baseCtx, spanCtx)
			ctx, span = tracer.Start(ctx, "Controller.Converse")
		} else {
			// Fallback to auto-generated trace ID if message_id is not a valid UUID
			ctx, span = tracer.Start(baseCtx, "Controller.Converse")
		}

		span.SetAttributes(
			attribute.String("project_id", projectIDStr),
			attribute.String("agent_name", agentName),
			attribute.String("namespace", reqPayload.Namespace),
			attribute.String("message_id", reqPayload.MessageID),
			attribute.String("session_id", reqPayload.SessionID),
		)

		// Fetch project to get default key
		_, projectDbSpan := tracer.Start(ctx, "DB.GetProjectByID")
		project, err := svc.Project.GetByID(ctx, projectID)
		if err != nil {
			projectDbSpan.RecordError(err)
			projectDbSpan.End()
			span.End()
			writeError(reqCtx, ctx, "Failed to fetch project", perrors.NewErrInternalServerError("Failed to fetch project", err))
			return
		}
		projectDbSpan.End()

		// Fetch agent configuration (includes model information)
		_, dbSpan := tracer.Start(ctx, "DB.GetAgentByName")
		agentConfig, err := svc.Agent.GetByName(ctx, projectID, agentName)
		if err != nil {
			dbSpan.RecordError(err)
			dbSpan.End()
			span.End()
			writeError(reqCtx, ctx, "Failed to fetch agent", perrors.NewErrInternalServerError("Failed to fetch agent", err))
			return
		}
		dbSpan.End()

		// Get default API key for the provider type
		providerType := llm.ProviderName(*agentConfig.ModelProviderType)

		reqHeaders := map[string]string{}
		reqCtx.Request.Header.VisitAll(func(key, value []byte) {
			reqHeaders[strings.ReplaceAll(string(key), "-", "_")] = string(value)
		})

		contextData := map[string]any{
			"Env":     utils.EnvironmentVariables(),
			"Context": reqPayload.Context,
			"Header":  reqHeaders,
		}

		// Get virtual key: prefer project's default key, fallback to header
		virtualKey := ""
		if project.DefaultKey != nil && *project.DefaultKey != "" {
			virtualKey = *project.DefaultKey
		} else {
			virtualKey = string(reqCtx.Request.Header.Peek("x-virtual-key"))
		}

		if virtualKey == "" {
			span.End()
			writeError(reqCtx, ctx, "Virtual key is required", perrors.NewErrInvalidRequest("Virtual key is required. Either set a default key for the project or provide x-virtual-key header", errors.New("virtual key is required")))
			return
		}

		// Parse model parameters from model configuration - directly unmarshal into ModelParameters
		var modelParams *responses.Parameters
		if agentConfig.ModelParameters != nil {
			if err := json.Unmarshal(*agentConfig.ModelParameters, &modelParams); err != nil {
				span.End()
				writeError(reqCtx, ctx, "Unable to parse model parameters", perrors.NewErrInvalidRequest("Model parameter is invalid", err))
				return
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
		mcpServerCacheMap := make(map[uuid.UUID]*tools.MCPServer)

		for _, agentMCP := range agentConfig.MCPServers {
			cacheKey := fmt.Sprintf("%s/%s", reqPayload.SessionID, agentMCP.MCPServerID)
			if cachedMcpServer, exists := mcpServerCache.Get(cacheKey); exists {
				mcpServer := cachedMcpServer.(*tools.MCPServer)
				if err = mcpServer.Client.Ping(ctx); err == nil {
					mcpServerCacheMap[agentMCP.MCPServerID] = mcpServer
					continue
				} else {
					mcpServerCache.Delete(cacheKey)
				}
			}
			mcpServerIDsToFetch = append(mcpServerIDsToFetch, agentMCP.MCPServerID)
		}

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
				for _, mcpServerID := range mcpServerIDsToFetch {
					mcpServerConfig, exists := mcpServerConfigs[mcpServerID]
					if !exists {
						continue
					}

					// Trace MCP server connection
					_, mcpConnSpan := tracer.Start(ctx, "MCP.Connect")
					mcpConnSpan.SetAttributes(
						attribute.String("mcp.server_id", mcpServerID.String()),
						attribute.String("mcp.endpoint", mcpServerConfig.Endpoint),
					)

					headers := make(map[string]string)
					if mcpServerConfig.Headers != nil {
						for k, v := range mcpServerConfig.Headers {
							headers[k] = utils.TryAndParseAsTemplate(v, contextData)
						}
					}

					mcpServer, err := tools.NewMCPServer(context.Background(), mcpServerConfig.Endpoint, headers)
					if err != nil {
						mcpConnSpan.RecordError(err)
						mcpConnSpan.End()
						slog.WarnContext(ctx, "Failed to connect to MCP server", slog.String("server_id", mcpServerID.String()), slog.Any("error", err))
						continue
					}
					mcpConnSpan.End()

					cacheKey := fmt.Sprintf("%s/%s", reqPayload.SessionID, mcpServerID)
					mcpServerCache.Set(cacheKey, mcpServer)
					mcpServerCacheMap[mcpServerID] = mcpServer
				}
			}
		}

		allTools := []core.Tool{}
		for _, agentMCP := range agentConfig.MCPServers {
			mcpServer, exists := mcpServerCacheMap[agentMCP.MCPServerID]
			if !exists {
				continue
			}

			var toolFilters []string
			if len(agentMCP.ToolFilters) > 0 {
				toolFilters = agentMCP.ToolFilters
			}

			tools := mcpServer.GetTools(toolFilters...)
			allTools = append(allTools, tools...)
		}

		promptLabel := "latest"
		if agentConfig.PromptLabel != nil && *agentConfig.PromptLabel != "" {
			promptLabel = *agentConfig.PromptLabel
		}

		reqCtx.Response.Header.Set("Content-Type", "text/event-stream")
		reqCtx.Response.Header.Set("Cache-Control", "no-cache")
		reqCtx.SetStatusCode(fasthttp.StatusOK)

		instructionProvider := prompts.NewWithLoader(
			adapters.NewInternalPromptPersistence(svc.Prompt, projectID, agentConfig.PromptName, promptLabel),
			prompts.WithDefaultResolver(contextData),
		)

		conversationManagerOpts := []history.ConversationManagerOptions{
			history.WithMessageID(reqPayload.MessageID),
		}

		var summarizer core.HistorySummarizer
		if agentConfig.EnableHistory && agentConfig.SummarizerType != nil && *agentConfig.SummarizerType != "none" {
			switch *agentConfig.SummarizerType {
			case "llm":
				if agentConfig.LLMSummarizerModelID == nil {
					span.End()
					writeError(reqCtx, ctx, "LLM summarizer model ID is required", perrors.NewErrInvalidRequest("llm_summarizer_model_id is required when summarizer_type is 'llm'", errors.New("llm_summarizer_model_id is required")))
					return
				}
				if agentConfig.LLMSummarizerPromptID == nil {
					span.End()
					writeError(reqCtx, ctx, "LLM summarizer prompt ID is required", perrors.NewErrInvalidRequest("llm_summarizer_prompt_id is required when summarizer_type is 'llm'", errors.New("llm_summarizer_prompt_id is required")))
					return
				}

				if agentConfig.SummarizerPromptName == nil || *agentConfig.SummarizerPromptName == "" {
					span.End()
					writeError(reqCtx, ctx, "Summarizer prompt name is missing", perrors.NewErrInternalServerError("Summarizer prompt name is missing", errors.New("summarizer prompt name missing")))
					return
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

				if agentConfig.SummarizerModelModelID == nil || agentConfig.SummarizerProviderType == nil {
					span.End()
					writeError(reqCtx, ctx, "Summarizer model configuration is incomplete", perrors.NewErrInternalServerError("Summarizer model or provider information is missing", errors.New("summarizer configuration incomplete")))
					return
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
					LLM:                 summarizerLLM,
					InstructionProvider: summarizerInstructionProvider,
					TokenThreshold:      tokenThreshold,
					KeepRecentCount:     keepRecentCount,
					Parameters:          summarizerModelParams,
				})

			case "sliding_window":
				if agentConfig.SlidingWindowKeepCount == nil || *agentConfig.SlidingWindowKeepCount <= 0 {
					span.End()
					writeError(reqCtx, ctx, "Sliding window keep count is required", perrors.NewErrInvalidRequest("sliding_window_keep_count is required and must be greater than 0 when summarizer_type is 'sliding_window'", errors.New("sliding_window_keep_count is required")))
					return
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

		allTools = append(allTools, tools.NewImageGenerationTool())

		agentOpts := &agents.AgentOptions{
			Name:                agentConfig.Name,
			LLM:                 llmClient,
			Tools:               allTools,
			InstructionProvider: instructionProvider,
			Parameters:          *modelParams,
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
				reqPayload.Namespace,
				reqPayload.PreviousMessageID,
				conversationManagerOpts...,
			)
		}

		agent := agents.NewAgent(agentOpts)

		reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
			// End the controller span when streaming completes
			defer span.End()
			defer w.Flush()

			_, execErr := agent.Execute(
				ctx,
				[]responses.InputMessageUnion{reqPayload.Message},
				func(message *responses.ResponseChunk) {
					buf, err := json.Marshal(message)
					if err != nil {
						return
					}

					_, _ = fmt.Fprintf(w, "event: %s\n", message.ChunkType())
					_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)
					_ = w.Flush()
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
