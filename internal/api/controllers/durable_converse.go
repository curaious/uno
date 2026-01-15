package controllers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
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
		log.Fatalf("failed to connect to redis: %v", err)
	}

	return cli
}

func RegisterDurableConverseRoute(r *router.Router, svc *services.Services, llmGateway *gateway.LLMGateway, conf *config.Config, broker core.StreamBroker) {
	temporalClient := getTemporalClient(conf)

	r.POST("/api/agent-server/converse", func(reqCtx *fasthttp.RequestCtx) {
		ctx, span := tracer.Start(reqCtx, "Controller.Converse")
		traceID := span.SpanContext().TraceID().String()
		reqCtx.Response.Header.Set("X-Trace-Id", traceID)

		projectIDStr := string(reqCtx.QueryArgs().Peek("project_id"))
		if projectIDStr == "" {
			writeError(reqCtx, ctx, "Project ID is required", perrors.NewErrInvalidRequest("project_id parameter is required", errors.New("project_id parameter is required")))
			return
		}

		//agentIDStr := string(reqCtx.QueryArgs().Peek("agent_id"))
		//if agentIDStr == "" {
		//	writeError(reqCtx, baseCtx, "Agent ID is required", perrors.NewErrInvalidRequest("agent_id parameter is required", errors.New("agent_id parameter is required")))
		//	return
		//}
		agentIDStr := "506d8e89-2190-466d-9524-83096bf122c6"

		agentName := string(reqCtx.QueryArgs().Peek("agent_name"))
		if agentName == "" {
			writeError(reqCtx, ctx, "Agent name is required", perrors.NewErrInvalidRequest("agent_name parameter is required", errors.New("agent_name parameter is required")))
			return
		}

		// Parse request body first to get message_id for trace ID
		var reqPayload ConverseRequest
		if err := json.Unmarshal(reqCtx.PostBody(), &reqPayload); err != nil {
			writeError(reqCtx, ctx, "Invalid request body", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		span.SetAttributes(
			attribute.String("project_id", projectIDStr),
			attribute.String("agent_name", agentName),
			attribute.String("namespace", reqPayload.Namespace),
			attribute.String("session_id", reqPayload.SessionID),
		)

		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			writeError(reqCtx, ctx, "unable to parse project id", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			writeError(reqCtx, ctx, "unable to parse agent id", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		agentConfig, err := svc.AgentConfig.GetByID(ctx, projectID, agentID)
		if err != nil {
			writeError(reqCtx, ctx, "unable to get agent config", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

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
		var stream <-chan *responses.ResponseChunk

		switch *agentConfig.Config.Runtime {
		case "Restate":
			runID = uuid.New().String()
			stream, err = broker.Subscribe(ctx, runID)
			if err != nil {
				fmt.Println("Error subscribing to stream for run ID:", runID, "error:", err)
				return
			}
			go streamChunks(ctx, reqCtx, stream)
			restateClient := ingress.NewClient(conf.RESTATE_SERVER_ENDPOINT)
			_, err = ingress.Workflow[*restate_agent_builder.WorkflowInput, *agents.AgentOutput](
				restateClient,
				"AgentBuilder",
				runID,
				"BuildAndExecuteAgent",
			).Request(ctx, &restate_agent_builder.WorkflowInput{
				AgentConfig: agentConfig,
				Input:       in,
			})
			if err != nil {
				writeError(reqCtx, ctx, "unable to create agent", perrors.NewErrInternalServerError(err.Error(), err))
				return
			}

		case "Temporal":
			run, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				TaskQueue: "AgentBuilder",
			}, "AgentBuilder", agentConfig, in)
			if err != nil {
				return
			}
			runID = run.GetID()
			stream, err = broker.Subscribe(ctx, runID)
			if err != nil {
				fmt.Println("Error subscribing to stream for run ID:", runID, "error:", err)
				return
			}
			go streamChunks(ctx, reqCtx, stream)

		default:
			b := streaming.NewMemoryStreamBroker()
			stream, err = b.Subscribe(ctx, "default")
			if err != nil {
				fmt.Println("Error subscribing to stream for run ID:", runID, "error:", err)
				return
			}

			in.Callback = func(chunk *responses.ResponseChunk) {
				b.Publish(ctx, "default", chunk)
			}
			go streamChunks(ctx, reqCtx, stream)

			_, err = builder.NewAgentBuilder(svc, llmGateway, b).BuildAndExecuteAgent(ctx, agentConfig, in)
			if err != nil {
				return
			}
		}

	})
}

func streamChunks(ctx context.Context, reqCtx *fasthttp.RequestCtx, stream <-chan *responses.ResponseChunk) {
	reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer w.Flush()

		span := trace.SpanFromContext(ctx)
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

				if m.OfRunCompleted != nil {
					return
				}
			}
		}
	})
}
