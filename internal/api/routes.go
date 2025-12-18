package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/fasthttp/router"
	"github.com/praveen001/uno/internal/api/controllers"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/propagation"
)

var tracePropagator = propagation.TraceContext{}

func (s *Server) initNewRoutes() fasthttp.RequestHandler {
	r := router.New()

	r.GET("/api/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		_, _ = ctx.Write([]byte("OK"))
	})

	controllers.RegisterConversationRoutes(r, s.services)
	controllers.RegisterProjectRoutes(r, s.services)
	controllers.RegisterProviderRoutes(r, s.services)
	controllers.RegisterModelRoutes(r, s.services)
	controllers.RegisterAgentRoutes(r, s.services)
	controllers.RegisterMCPServerRoutes(r, s.services)
	controllers.RegisterPromptRoutes(r, s.services)
	controllers.RegisterSchemaRoutes(r, s.services)
	controllers.RegisterVirtualKeyRoutes(r, s.services)
	controllers.RegisterGatewayRoutes(r.Group("/api/gateway"), s.services)
	controllers.RegisterConverseRoute(r, s.services)
	controllers.RegisterTracesRoutes(r.Group("/api/agent-server"), s.services)
	controllers.RegisterDurableConverseRoute(r, s.services)

	return s.withMiddlewares(r.Handler)
}

func (s *Server) withMiddlewares(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		applyCORS(ctx)
		if string(ctx.Method()) == fasthttp.MethodOptions {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		start := time.Now()
		uri := ctx.URI()
		requestURI := string(uri.FullURI())
		slog.Info("Started processing", slog.String("method", string(ctx.Method())), slog.String("request_uri", requestURI))

		h := http.Header{}
		ctx.Request.Header.VisitAll(func(k, v []byte) {
			h[string(k)] = []string{string(v)}
		})
		traceCtx := tracePropagator.Extract(ctx, propagation.HeaderCarrier(h))
		ctx.SetUserValue("traceCtx", traceCtx)
		next(ctx)

		slog.Info("Finished processing", slog.String("method", string(ctx.Method())), slog.String("request_uri", requestURI), slog.Duration("duration", time.Since(start)))
	}
}

func applyCORS(ctx *fasthttp.RequestCtx) {
	headers := &ctx.Response.Header
	headers.Set("Access-Control-Allow-Origin", "*")
	headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
	headers.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Planner-Amg-Id, X-Planner-Channel-Id, X-Service-Id")
}
