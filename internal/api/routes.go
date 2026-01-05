package api

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/praveen001/uno/internal/api/authenticator"
	"github.com/praveen001/uno/internal/api/controllers"
	"github.com/praveen001/uno/internal/config"
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

	conf := config.ReadConfig()
	auth, err := authenticator.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	controllers.RegisterAuthRoutes(r, s.services, auth)
	controllers.RegisterConversationRoutes(r, s.services)
	controllers.RegisterProjectRoutes(r, s.services)
	controllers.RegisterProviderRoutes(r, s.services)
	controllers.RegisterModelRoutes(r, s.services)
	controllers.RegisterAgentRoutes(r, s.services)
	controllers.RegisterMCPServerRoutes(r, s.services)
	controllers.RegisterPromptRoutes(r, s.services)
	controllers.RegisterSchemaRoutes(r, s.services)
	controllers.RegisterVirtualKeyRoutes(r, s.services)
	controllers.RegisterGatewayRoutes(r.Group("/api/gateway"), s.services, s.llmGateway)
	controllers.RegisterConverseRoute(r, s.services, s.llmGateway)
	controllers.RegisterTracesRoutes(r.Group("/api/agent-server"), s.services)
	controllers.RegisterDurableConverseRoute(r, s.services)

	return s.withMiddlewares(r.Handler, auth)
}

func (s *Server) withMiddlewares(next fasthttp.RequestHandler, auth *authenticator.Authenticator) fasthttp.RequestHandler {
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

		// Auth check
		if auth.AuthEnabled() && !isPublicRoute(ctx) {
			accessToken := strings.TrimPrefix(string(ctx.Request.Header.Peek("Authorization")), "Bearer ")
			if accessToken == "" {
				accessToken = string(ctx.Request.Header.Cookie("access_token"))
			}

			if accessToken == "" {
				ctx.SetStatusCode(fasthttp.StatusUnauthorized)
				return
			}

			claims, err := auth.VerifyAccessToken(ctx, accessToken)
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusUnauthorized)
				return
			}

			// Store user claims in context for downstream handlers
			ctx.SetUserValue("userClaims", claims)
		}

		next(ctx)

		slog.Info("Finished processing", slog.String("method", string(ctx.Method())), slog.String("request_uri", requestURI), slog.Duration("duration", time.Since(start)))
	}
}

func applyCORS(ctx *fasthttp.RequestCtx) {
	headers := &ctx.Response.Header
	headers.Set("Access-Control-Allow-Origin", string(ctx.Request.Header.Peek("Origin")))
	headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
	headers.Set("Access-Control-Allow-Headers", os.Getenv("ALLOWED_HEADERS"))
	headers.Set("Access-Control-Allow-Credentials", "true")
}

func isPublicRoute(ctx *fasthttp.RequestCtx) bool {
	path := string(ctx.Path())

	// Public auth routes
	publicAuthRoutes := []string{
		"/api/agent-server/auth/login",
		"/api/agent-server/auth/callback",
		"/api/agent-server/auth/enabled",
	}

	switch {
	case path == "/api/health":
		return true
	default:
		for _, route := range publicAuthRoutes {
			if path == route {
				return true
			}
		}
		return false
	}
}
