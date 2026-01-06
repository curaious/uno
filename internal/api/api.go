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
	"github.com/curaious/uno/internal/pubsub"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/gateway/middlewares/logger"
	"github.com/curaious/uno/pkg/gateway/middlewares/virtual_key_middleware"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"

	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/migrations"
)

// Server is an HTTP Server with access to *pm.App
type Server struct {
	srv         *fasthttp.Server
	addr        string
	services    *services.Services
	configStore *adapters.ServiceConfigStore
	llmGateway  *gateway.LLMGateway
	pubsub      *pubsub.PubSub
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

	s := &Server{
		srv:         &fasthttp.Server{},
		addr:        fmt.Sprintf("0.0.0.0:6060"),
		services:    svc,
		configStore: configStore,
		llmGateway:  llmGateway,
		pubsub:      ps,
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
