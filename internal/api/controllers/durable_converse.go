package controllers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	json "github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/agent_builder/builder"
	"github.com/curaious/uno/internal/agent_builder/restate_agent_builder"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/perrors"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/streaming"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
)

var (
	tracer = otel.Tracer("Controller")
)

type ConverseRequest struct {
	Message           responses.InputMessageUnion `json:"message" doc:"User message"`
	Namespace         string                      `json:"namespace" doc:"Namespace ID"`
	PreviousMessageID string                      `json:"previous_message_id" doc:"Previous run ID for threading"`
	Context           map[string]any              `json:"context" doc:"Context to pass to prompt template"`
	SessionID         string                      `json:"session_id" required:"true" doc:"Session ID"`
}

func getTemporalClient(conf *config.Config) client.Client {
	otelInterceptor, err := opentelemetry.NewTracingInterceptor(
		opentelemetry.TracerOptions{},
	)
	if err != nil {
		panic(err)
	}

	cli, err := client.Dial(client.Options{
		HostPort: conf.TEMPORAL_SERVER_HOST_PORT,
		Interceptors: []interceptor.ClientInterceptor{
			otelInterceptor,
		},
	})
	if err != nil {
		log.Fatalf("failed to connect to temporal: %v", err)
	}

	return cli
}

func RecordSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func RegisterDurableConverseRoute(r *router.Router, svc *services.Services, llmGateway *gateway.LLMGateway, conf *config.Config, broker core.StreamBroker) {
	var temporalClient client.Client
	if strings.Contains(conf.RUNTIME_ENABLED, "temporal") {
		temporalClient = getTemporalClient(conf)
	}

	var restateClient *ingress.Client
	if strings.Contains(conf.RUNTIME_ENABLED, "restate") {
		restateClient = ingress.NewClient(conf.RESTATE_SERVER_ENDPOINT)
	}

	r.POST("/api/agent-server/converse", func(reqCtx *fasthttp.RequestCtx) {
		ctx, span := tracer.Start(reqCtx, "Controller.Converse")
		traceID := span.SpanContext().TraceID().String()
		reqCtx.Response.Header.Set("X-Trace-Id", traceID)

		projectIDStr := string(reqCtx.QueryArgs().Peek("project_id"))
		if projectIDStr == "" {
			RecordSpanError(span, errors.New("project_id is required"))
			writeError(reqCtx, ctx, "Project ID is required", perrors.NewErrInvalidRequest("project_id parameter is required", errors.New("project_id parameter is required")))
			return
		}

		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			RecordSpanError(span, err)
			writeError(reqCtx, ctx, "unable to parse project id", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		agentIDStr := string(reqCtx.QueryArgs().Peek("agent_id"))
		if agentIDStr == "" {
			RecordSpanError(span, errors.New("agent_id is required"))
			writeError(reqCtx, ctx, "Agent ID is required", perrors.NewErrInvalidRequest("agent_id parameter is required", errors.New("agent_id parameter is required")))
			return
		}

		version := 0
		frag := strings.Split(agentIDStr, ":")

		agentID, err := uuid.Parse(frag[0])
		if err != nil {
			RecordSpanError(span, err)
			writeError(reqCtx, ctx, "unable to parse agent id", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		if len(frag) > 1 {
			agentIDStr = frag[0]
			v, err := strconv.Atoi(frag[1])
			if err != nil {
				v, err = svc.AgentConfig.GetAgentVersionByAlias(ctx, projectID, agentID, frag[1])
			}
			if err != nil {
				RecordSpanError(span, err)
				writeError(reqCtx, ctx, "invalid version or alias", perrors.NewErrInternalServerError(err.Error(), err))
				return
			}
			version = v
		}

		// Parse request body first to get message_id for trace ID
		var reqPayload ConverseRequest
		if err := json.Unmarshal(reqCtx.PostBody(), &reqPayload); err != nil {
			RecordSpanError(span, err)
			writeError(reqCtx, ctx, "Invalid request body", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		span.SetAttributes(
			attribute.String("project_id", projectIDStr),
			attribute.String("agent_id", agentIDStr),
			attribute.String("namespace", reqPayload.Namespace),
			attribute.String("session_id", reqPayload.SessionID),
		)

		project, err := svc.Project.GetByID(ctx, projectID)
		if err != nil {
			RecordSpanError(span, err)
			writeError(reqCtx, ctx, "unable to get project", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		// TODO: avoid default key, and come up with a better mechanism
		if project.DefaultKey == nil {
			RecordSpanError(span, errors.New("project default key is required"))
			writeError(reqCtx, ctx, "project default key is required", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		agentConfig, err := svc.AgentConfig.GetByAgentIDAndVersion(ctx, projectID, agentID, version)
		if err != nil {
			RecordSpanError(span, err)
			writeError(reqCtx, ctx, "unable to get agent config", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}
		span.SetAttributes(attribute.String("agent_name", agentConfig.Name))

		reqHeaders := map[string]string{}
		reqCtx.Request.Header.VisitAll(func(key, value []byte) {
			reqHeaders[strings.ReplaceAll(string(key), "-", "_")] = string(value)
		})

		contextData := map[string]any{
			"Env":     utils.EnvironmentVariables(),
			"Context": reqPayload.Context,
			"Header":  reqHeaders,
		}

		in := &agents.AgentInput{
			Namespace:         reqPayload.Namespace,
			PreviousMessageID: reqPayload.PreviousMessageID,
			Messages:          []responses.InputMessageUnion{reqPayload.Message},
			RunContext:        contextData,
		}

		var runID string

		reqCtx.Response.Header.Set("Content-Type", "text/event-stream")
		reqCtx.Response.Header.Set("Cache-Control", "no-cache")
		reqCtx.SetStatusCode(fasthttp.StatusOK)

		switch *agentConfig.Config.Runtime {
		case "Restate":
			if restateClient == nil {
				err = errors.New("restate runtime is not enabled")
				RecordSpanError(span, err)
				writeError(reqCtx, ctx, err.Error(), perrors.NewErrInternalServerError(err.Error(), err))
				return
			}
			runID = uuid.New().String()
			// Subscribe first to ensure we don't miss any chunks
			stream, subErr := broker.Subscribe(ctx, runID)
			if subErr != nil {
				RecordSpanError(span, subErr)
				writeError(reqCtx, ctx, "failed to subscribe to stream", perrors.NewErrInternalServerError(subErr.Error(), subErr))
				return
			}

			// Start workflow in goroutine so the handler can return and streaming can begin
			go func() {
				_, err := ingress.Workflow[*restate_agent_builder.WorkflowInput, *agents.AgentOutput](
					restateClient,
					"AgentBuilder",
					runID,
					"BuildAndExecuteAgent",
				).Request(ctx, &restate_agent_builder.WorkflowInput{
					AgentConfig: agentConfig,
					Input:       in,
					Key:         *project.DefaultKey,
				})
				if err != nil {
					RecordSpanError(span, err)
				}
			}()

			// Stream chunks - this allows the handler to return so streaming can start
			streamChunksFromChannel(ctx, reqCtx, stream, span)

		case "Temporal":
			if temporalClient == nil {
				err = errors.New("restate runtime is not enabled")
				RecordSpanError(span, err)
				writeError(reqCtx, ctx, err.Error(), perrors.NewErrInternalServerError(err.Error(), err))
				return
			}

			runID = uuid.New().String()
			// Subscribe first to ensure we don't miss any chunks
			stream, subErr := broker.Subscribe(ctx, runID)
			if subErr != nil {
				RecordSpanError(span, subErr)
				writeError(reqCtx, ctx, "failed to subscribe to stream", perrors.NewErrInternalServerError(subErr.Error(), subErr))
				return
			}

			// Start workflow in goroutine so the handler can return and streaming can begin
			go func() {
				run, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
					ID:        runID,
					TaskQueue: "AgentBuilder",
				}, "AgentBuilder", agentConfig, in, project.DefaultKey)
				if err != nil {
					RecordSpanError(span, err)
					return
				}

				var output agents.AgentOutput
				err = run.Get(ctx, &output)
				if err != nil {
					RecordSpanError(span, err)
				}
			}()

			// Stream chunks - this allows the handler to return so streaming can start
			streamChunksFromChannel(ctx, reqCtx, stream, span)

		default:
			b := streaming.NewMemoryStreamBroker()
			// Subscribe first to ensure we don't miss any chunks
			stream, subErr := b.Subscribe(ctx, "default")
			if subErr != nil {
				RecordSpanError(span, subErr)
				writeError(reqCtx, ctx, "failed to subscribe to stream", perrors.NewErrInternalServerError(subErr.Error(), subErr))
				return
			}

			in.Callback = func(chunk *responses.ResponseChunk) {
				b.Publish(ctx, "default", chunk)
			}

			// Start execution in goroutine so the handler can return and streaming can begin
			go func() {
				defer b.Close(ctx, "default")
				_, err := builder.NewAgentBuilder(svc, llmGateway, b).BuildAndExecuteAgent(ctx, agentConfig, in, *project.DefaultKey)
				if err != nil {
					RecordSpanError(span, err)
				}
			}()

			// Stream chunks - this allows the handler to return so streaming can start
			streamChunksFromChannel(ctx, reqCtx, stream, span)
		}

	})
}

// streamChunksFromChannel sets up SSE streaming from a pre-subscribed channel.
// The channel must be subscribed BEFORE the workflow starts to avoid missing chunks.
// This function sets up SetBodyStreamWriter and returns immediately, allowing
// the HTTP handler to return so fasthttp can begin streaming the response.
func streamChunksFromChannel(ctx context.Context, reqCtx *fasthttp.RequestCtx, stream <-chan *responses.ResponseChunk, span trace.Span) {
	reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer w.Flush()
		defer span.End()

		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-stream:
				if !ok {
					return
				}

				buf, _ := json.Marshal(m)

				_, _ = fmt.Fprintf(w, "event: %s\n", m.ChunkType())
				_, _ = fmt.Fprintf(w, "data: %s\n\n", string(buf))
				_ = w.Flush()

				if m.OfRunCompleted != nil || m.OfRunPaused != nil {
					return
				}
			}
		}
	})
}
