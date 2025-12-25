package controllers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"

	json "github.com/bytedance/sonic"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/config"
	"github.com/praveen001/uno/internal/perrors"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/service"
	"github.com/redis/go-redis/v9"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func RegisterDurableConverseRoute(r *router.Router, svc *services.Services) {
	conf := config.ReadConfig()
	r.POST("/api/agent-server/converse2", func(reqCtx *fasthttp.RequestCtx) {
		baseCtx := requestContext(reqCtx)

		projectIDStr := string(reqCtx.QueryArgs().Peek("project_id"))
		if projectIDStr == "" {
			writeError(reqCtx, baseCtx, "Project ID is required", perrors.NewErrInvalidRequest("project_id parameter is required", errors.New("project_id parameter is required")))
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

		in := service.AgentRunInput{
			Message:           reqPayload.Message,
			Namespace:         reqPayload.Namespace,
			PreviousMessageID: reqPayload.PreviousMessageID,
			Context:           reqPayload.Context,
			SessionID:         reqPayload.SessionID,
			ProjectID:         projectIDStr,
			AgentName:         agentName,
		}

		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", conf.REDIS_HOST, conf.REDIS_PORT),
			DB:       10,
			Username: conf.REDIS_USERNAME,
			Password: conf.REDIS_PASSWORD,
		})

		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			slog.Error("failed to connect to redis", "error", err)
			return
		}

		s := trace.SpanFromContext(ctx)
		fmt.Println(s.SpanContext().IsValid())

		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{}, // W3C traceparent
				propagation.Baggage{},
			),
		)

		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)

		ctx, cancel := context.WithCancel(ctx)
		channel := "stream:" + uuid.New().String()
		ps := redisClient.Subscribe(reqCtx, channel)

		restateClient := ingress.NewClient("http://localhost:8081")

		go func() {
			defer cancel()
			// To call a service
			_, err := ingress.Workflow[*service.AgentRunInput, *service.AgentRunOutput](
				restateClient, "AgentWorkflow", channel, "Run").
				Request(ctx, &in, restate.WithHeaders(carrier))
			if err != nil {
				return
			}
		}()

		reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
			// End the controller span when streaming completes
			defer w.Flush()
			defer span.End()

			for {
				select {
				case <-ctx.Done():
					return
				case m := <-ps.Channel():
					_, _ = fmt.Fprintf(w, "data: %s\n\n", m.Payload)
					_ = w.Flush()
				}
			}
		})
	})
}
