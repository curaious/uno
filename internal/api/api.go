package api

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curaious/uno/internal/adapters"
	"github.com/curaious/uno/internal/agent_builder/restate_agent_builder"
	"github.com/curaious/uno/internal/agent_builder/temporal_agent_builder"
	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/migrations"
	"github.com/curaious/uno/internal/pubsub"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/streaming"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/gateway/middlewares/logger"
	"github.com/curaious/uno/pkg/gateway/middlewares/virtual_key_middleware"
	"github.com/curaious/uno/pkg/sandbox"
	"github.com/curaious/uno/pkg/sandbox/docker_sandbox"
	"github.com/curaious/uno/pkg/sandbox/k8s_sandbox"
	"github.com/redis/go-redis/v9"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
	"github.com/valyala/fasthttp"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Server is an HTTP Server with access to *pm.App
type Server struct {
	conf          *config.Config
	srv           *fasthttp.Server
	addr          string
	services      *services.Services
	llmGateway    *gateway.LLMGateway
	pubsub        *pubsub.PubSub
	redisClient   *redis.Client
	broker        core.StreamBroker
	sandboxManger sandbox.Manager
}

// New creates a new server by wrapping *planner.App with *http.Server
func New() *Server {
	conf := config.ReadConfig()

	m, err := migrations.NewMigrator()
	if err != nil {
		panic("unable to create migrator")
	}

	err = m.Up(0)
	if err != nil {
		panic("unable to run migrations")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conf.REDIS_HOST, conf.REDIS_PORT),
		DB:       10,
		Username: conf.REDIS_USERNAME,
		Password: conf.REDIS_PASSWORD,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	svc := services.NewServices(conf)
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

	// Create shared LLM gateway
	llmGateway := gateway.NewLLMGateway(configStore)
	llmGateway.UseMiddleware(
		logger.NewLoggerMiddleware(),
		virtual_key_middleware.NewVirtualKeyMiddleware(
			configStore,
			virtual_key_middleware.NewRedisRateLimiterStorage(redisClient, ""),
		),
	)
	slog.Info("LLM gateway initialized with pubsub")

	// Broker
	broker, err := streaming.NewRedisStreamBroker(streaming.RedisStreamBrokerOptions{
		Client: redisClient,
	})
	if err != nil {
		log.Fatalf("Failed to create redis stream broker: %v", err)
	}
	slog.Info("Redis stream broker initialized")

	// Sandbox manager
	var sandboxManager sandbox.Manager
	if config.GetEnvOrDefault("SANDBOX_ENABLED", "false") == "true" {
		agentDataPath := conf.GetAgentDataPath()
		if err := os.MkdirAll(agentDataPath, 0755); err != nil {
			slog.Warn("Failed to create sandbox data directory", slog.String("path", agentDataPath), slog.Any("error", err))
		}

		sandboxManager, _ = k8s_sandbox.NewManager(k8s_sandbox.Config{
			AgentDataPath: agentDataPath,
		})

		if sandboxManager == nil {
			sandboxManager = docker_sandbox.NewManager(docker_sandbox.Config{
				AgentDataPath:   agentDataPath,
				SessionDataPath: conf.GetSessionDataPath(),
			})
		}

		slog.Info("Sandbox manager initialized", slog.String("agent_data_path", agentDataPath))
	}

	s := &Server{
		conf:          conf,
		srv:           &fasthttp.Server{},
		addr:          fmt.Sprintf("0.0.0.0:6060"),
		services:      svc,
		llmGateway:    llmGateway,
		pubsub:        ps,
		redisClient:   redisClient,
		broker:        broker,
		sandboxManger: sandboxManager,
	}

	s.srv.Handler = s.initNewRoutes()

	return s
}

// Start the rest server
func (s *Server) Start() {
	slog.Info("Starting REST server...")
	go func() {
		if err := s.srv.ListenAndServe(s.addr); err != nil {
			slog.Error("Server shutdown", slog.Any("error", err))
		}
	}()
	slog.Info("REST server started!")

	// Listen for OS interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block till we receive an interrupt
	<-c
	slog.Info("Received interrupt...")

	// Create a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	s.shutdown(ctx)
}

// StartTemporalWorker the temporal worker
func (s *Server) StartTemporalWorker() {
	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		log.Fatalln("Unable to create interceptor", err)
	}

	cli, err := client.Dial(client.Options{
		HostPort:     s.conf.TEMPORAL_SERVER_HOST_PORT,
		Interceptors: []interceptor.ClientInterceptor{tracingInterceptor},
	})
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer cli.Close()

	agentBuilder := temporal_agent_builder.NewAgentBuilder(s.services, s.llmGateway, s.broker, s.sandboxManger)

	w := worker.New(cli, "AgentBuilder", worker.Options{})

	w.RegisterActivityWithOptions(agentBuilder.GetPrompt, activity.RegisterOptions{Name: "GetPrompt"})
	w.RegisterActivityWithOptions(agentBuilder.LLMCall, activity.RegisterOptions{Name: "LLMCall"})
	w.RegisterActivityWithOptions(agentBuilder.LoadMessages, activity.RegisterOptions{Name: "LoadMessages"})
	w.RegisterActivityWithOptions(agentBuilder.SaveMessages, activity.RegisterOptions{Name: "SaveMessages"})
	w.RegisterActivityWithOptions(agentBuilder.SaveSummary, activity.RegisterOptions{Name: "SaveSummary"})
	w.RegisterActivityWithOptions(agentBuilder.Summarize, activity.RegisterOptions{Name: "Summarize"})
	w.RegisterActivityWithOptions(agentBuilder.MCPListTools, activity.RegisterOptions{Name: "MCPListTools"})
	w.RegisterActivityWithOptions(agentBuilder.MCPCallTool, activity.RegisterOptions{Name: "MCPCallTool"})
	w.RegisterActivityWithOptions(agentBuilder.SandboxTool, activity.RegisterOptions{Name: "SandboxTool"})

	w.RegisterWorkflowWithOptions(agentBuilder.BuildAndExecuteAgent, workflow.RegisterOptions{
		Name: "AgentBuilder",
	})

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("failed to run agent builder: %v", err)
	}
}

// StartRestateWorker the restate worker
func (s *Server) StartRestateWorker() {
	agentBuilder := restate_agent_builder.NewAgentBuilder(s.services, s.llmGateway, s.broker, s.sandboxManger)

	if err := server.NewRestate().
		Bind(restate.Reflect(agentBuilder)).
		Start(context.Background(), ":9080"); err != nil {
		log.Fatal(err)
	}
}

// Shutdown shuts down the rest server
func (s *Server) shutdown(ctx context.Context) {
	slog.Info("Gracefully shutting down REST server...")

	// Stop pubsub listener
	if s.pubsub != nil {
		s.pubsub.Stop()
	}

	if err := s.srv.Shutdown(); err != nil {
		slog.Error("Failed to shutdown the server", slog.Any("error", err))
	}
	slog.Info("REST server shutdown!")
}
