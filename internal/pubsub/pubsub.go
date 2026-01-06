package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/curaious/uno/internal/config"
	"github.com/lib/pq"
)

// ConfigChangeType represents the type of configuration change
type ConfigChangeType string

const (
	ChangeTypeProviderConfig     ConfigChangeType = "provider_configs"
	ChangeTypeAPIKey             ConfigChangeType = "api_keys"
	ChangeTypeVirtualKey         ConfigChangeType = "virtual_keys"
	ChangeTypeVirtualKeyProvider ConfigChangeType = "virtual_key_providers"
	ChangeTypeVirtualKeyModel    ConfigChangeType = "virtual_key_models"
)

// ConfigChangeEvent represents a configuration change notification
type ConfigChangeEvent struct {
	ChangeType ConfigChangeType
	Operation  string // INSERT, UPDATE, DELETE
}

// ConfigChangeHandler is a callback function for config changes
type ConfigChangeHandler func(event ConfigChangeEvent)

// PubSub handles PostgreSQL LISTEN/NOTIFY for configuration changes
type PubSub struct {
	connStr  string
	listener *pq.Listener
	handlers []ConfigChangeHandler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPubSub creates a new PubSub instance
func NewPubSub(conf *config.Config) *PubSub {
	connStr := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v",
		conf.DB_USERNAME, conf.DB_PASSWORD, conf.DB_HOST, conf.DB_PORT, conf.DB_NAME)
	if conf.DISABLE_TLS == "true" {
		connStr = connStr + "?sslmode=disable"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PubSub{
		connStr:  connStr,
		handlers: make([]ConfigChangeHandler, 0),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Subscribe adds a handler for config change events
func (ps *PubSub) Subscribe(handler ConfigChangeHandler) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.handlers = append(ps.handlers, handler)
}

// Start begins listening for notifications
func (ps *PubSub) Start() error {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			slog.Error("PubSub listener error", slog.Any("error", err))
		}
		if ev == pq.ListenerEventConnectionAttemptFailed {
			slog.Warn("PubSub connection attempt failed, will retry")
		}
		if ev == pq.ListenerEventDisconnected {
			slog.Warn("PubSub disconnected, will attempt reconnect")
		}
		if ev == pq.ListenerEventReconnected {
			slog.Info("PubSub reconnected, triggering full reload")
			// On reconnection, notify handlers to reload all data
			// since we might have missed notifications
			ps.notifyHandlers(ConfigChangeEvent{
				ChangeType: ChangeTypeProviderConfig,
				Operation:  "RELOAD",
			})
			ps.notifyHandlers(ConfigChangeEvent{
				ChangeType: ChangeTypeAPIKey,
				Operation:  "RELOAD",
			})
			ps.notifyHandlers(ConfigChangeEvent{
				ChangeType: ChangeTypeVirtualKey,
				Operation:  "RELOAD",
			})
		}
	}

	ps.listener = pq.NewListener(ps.connStr, 10*time.Second, time.Minute, reportProblem)

	if err := ps.listener.Listen("config_changes"); err != nil {
		return fmt.Errorf("failed to listen on config_changes channel: %w", err)
	}

	slog.Info("PubSub started listening for config changes")

	// Start the notification processing goroutine
	go ps.processNotifications()

	return nil
}

// Stop closes the listener
func (ps *PubSub) Stop() {
	ps.cancel()
	if ps.listener != nil {
		ps.listener.Close()
	}
	slog.Info("PubSub stopped")
}

func (ps *PubSub) processNotifications() {
	for {
		select {
		case <-ps.ctx.Done():
			return
		case notification := <-ps.listener.Notify:
			if notification == nil {
				// Connection lost, will be handled by reportProblem callback
				continue
			}

			// Parse the payload: "table_name:operation"
			parts := strings.SplitN(notification.Extra, ":", 2)
			if len(parts) != 2 {
				slog.Warn("Invalid notification payload", slog.String("payload", notification.Extra))
				continue
			}

			event := ConfigChangeEvent{
				ChangeType: ConfigChangeType(parts[0]),
				Operation:  parts[1],
			}

			slog.Debug("Received config change notification",
				slog.String("table", string(event.ChangeType)),
				slog.String("operation", event.Operation))

			ps.notifyHandlers(event)
		}
	}
}

func (ps *PubSub) notifyHandlers(event ConfigChangeEvent) {
	ps.mu.RLock()
	handlers := make([]ConfigChangeHandler, len(ps.handlers))
	copy(handlers, ps.handlers)
	ps.mu.RUnlock()

	for _, handler := range handlers {
		// Run handlers in goroutines to avoid blocking the notification loop
		go handler(event)
	}
}
