package controllers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	json "github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/agent_builder/restate_agent_builder"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/perrors"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/streaming"
	"github.com/curaious/uno/pkg/llm/responses"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/client"
)

func RegisterDurableConverseRoute(r *router.Router, svc *services.Services, conf *config.Config) {
	r.POST("/api/agent-server/converse", func(reqCtx *fasthttp.RequestCtx) {
		baseCtx := requestContext(reqCtx)

		projectIDStr := string(reqCtx.QueryArgs().Peek("project_id"))
		if projectIDStr == "" {
			writeError(reqCtx, baseCtx, "Project ID is required", perrors.NewErrInvalidRequest("project_id parameter is required", errors.New("project_id parameter is required")))
			return
		}

		//agentIDStr := string(reqCtx.QueryArgs().Peek("agent_id"))
		//if agentIDStr == "" {
		//	writeError(reqCtx, baseCtx, "Agent ID is required", perrors.NewErrInvalidRequest("agent_id parameter is required", errors.New("agent_id parameter is required")))
		//	return
		//}
		agentIDStr := "73b58e1f-7821-4d18-bddd-859dc7b65478"

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

		var ctx context.Context
		var span trace.Span
		ctx, span = tracer.Start(baseCtx, "Controller.Converse")
		traceID := span.SpanContext().TraceID().String()
		reqCtx.Response.Header.Set("X-Trace-Id", traceID)

		span.SetAttributes(
			attribute.String("project_id", projectIDStr),
			attribute.String("agent_name", agentName),
			attribute.String("namespace", reqPayload.Namespace),
			attribute.String("session_id", reqPayload.SessionID),
		)

		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			writeError(reqCtx, baseCtx, "unable to parse project id", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			writeError(reqCtx, baseCtx, "unable to parse agent id", perrors.NewErrInternalServerError(err.Error(), err))
			return
		}

		agentConfig, err := svc.AgentConfig.GetByID(baseCtx, projectID, agentID)
		if err != nil {
			writeError(reqCtx, baseCtx, "unable to get agent config", perrors.NewErrInternalServerError(err.Error(), err))
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

		switch *agentConfig.Config.Runtime {
		case "Restate":
			runID = uuid.New().String()
			restateClient := ingress.NewClient(conf.RESTATE_SERVER_ENDPOINT)
			_, err = ingress.Workflow[*restate_agent_builder.WorkflowInput, *agents.AgentOutput](
				restateClient,
				"AgentBuilder",
				runID,
				"BuildAndExecuteAgent",
			).Send(ctx, &restate_agent_builder.WorkflowInput{
				AgentConfig: agentConfig,
				Input:       in,
			})
			if err != nil {
				writeError(reqCtx, baseCtx, "unable to create agent", perrors.NewErrInternalServerError(err.Error(), err))
				return
			}

		case "Temporal":
			cli, err := client.Dial(client.Options{
				HostPort: conf.TEMPORAL_SERVER_HOST_PORT,
			})
			if err != nil {
				log.Fatalf("failed to connect to redis: %v", err)
			}
			run, err := cli.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				TaskQueue: "AgentBuilder",
			}, "AgentBuilder", agentConfig, in)
			if err != nil {
				return
			}
			runID = run.GetID()

		default:
			// TODO: local run time
		}

		reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
			// End the controller span when streaming completes
			defer w.Flush()
			defer span.End()
			defer func() {
				fmt.Println("### COMPLETED ###")
			}()

			redisClient := redis.NewClient(&redis.Options{
				Addr:     fmt.Sprintf("%s:%s", conf.REDIS_HOST, conf.REDIS_PORT),
				DB:       10,
				Username: conf.REDIS_USERNAME,
				Password: conf.REDIS_PASSWORD,
			})

			broker, err := streaming.NewRedisStreamBroker(streaming.RedisStreamBrokerOptions{
				Client: redisClient,
			})
			if err != nil {
				log.Fatalf("Failed to create redis stream broker: %v", err)
			}

			// Handle streaming via callback
			fmt.Println("Subscribing to stream for run ID:", runID)
			stream, err := broker.Subscribe(ctx, runID)
			if err != nil {
				fmt.Println("Error subscribing to stream for run ID:", runID, "error:", err)
				return
			}

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
	})
}
