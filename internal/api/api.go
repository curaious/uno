package api

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/praveen001/uno/internal/services"
	"github.com/valyala/fasthttp"

	"github.com/praveen001/uno/internal/config"
	"github.com/praveen001/uno/internal/migrations"
)

// Server is an HTTP Server with access to *pm.App
type Server struct {
	srv      *fasthttp.Server
	addr     string
	services *services.Services
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

	s := &Server{
		srv:      &fasthttp.Server{},
		addr:     fmt.Sprintf("0.0.0.0:6060"),
		services: services.NewServices(conf),
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
	if err := s.srv.Shutdown(); err != nil {
		slog.Error("Failed to shutdown the server", slog.Any("error", err))
	}
	slog.Info("REST server shutdown!")
}
