package daemon

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	defaultPort       = "8080"
	defaultSandboxDir = "/sandbox/workspace"

	// IdleTimeout is the duration after which the daemon will shut down if no requests are received.
	// Change this value to adjust how long the daemon waits before auto-terminating.
	IdleTimeout = 30 * time.Second
)

// idleTracker keeps track of the last request time to implement idle timeout
type idleTracker struct {
	mu            sync.Mutex
	lastRequestAt time.Time
}

func newIdleTracker() *idleTracker {
	return &idleTracker{
		lastRequestAt: time.Now(),
	}
}

func (t *idleTracker) touch() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastRequestAt = time.Now()
}

func (t *idleTracker) idleDuration() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return time.Since(t.lastRequestAt)
}

// withIdleTracking is a middleware that updates the last request time on every request
func withIdleTracking(tracker *idleTracker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.touch()
		next.ServeHTTP(w, r)
	})
}

// startIdleWatcher starts a goroutine that monitors idle time and exits the process if idle too long
func startIdleWatcher(tracker *idleTracker) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			idle := tracker.idleDuration()
			if idle >= IdleTimeout {
				log.Printf("sandbox-daemon idle for %v (timeout: %v), shutting down", idle, IdleTimeout)
				os.Exit(0)
			}
		}
	}()
}

func NewSandboxDaemon() {
	port := getenv("SANDBOX_PORT", defaultPort)
	root := getenv("SANDBOX_ROOT", defaultSandboxDir)

	tracker := newIdleTracker()

	mux := http.NewServeMux()

	// Exec endpoints
	mux.Handle("/exec/bash", withJSON(withSandboxRoot(root, handleExecBash)))
	mux.Handle("/exec/python", withJSON(withSandboxRoot(root, handleExecPython)))

	// File endpoints: /files/<path>
	mux.Handle("/files/", withJSON(withSandboxRoot(root, handleFiles)))

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Start the idle watcher
	startIdleWatcher(tracker)

	addr := ":" + port
	log.Printf("sandbox-daemon listening on %s (root=%s, idle_timeout=%v)", addr, filepath.Clean(root), IdleTimeout)

	// Wrap with otel HTTP server tracing (extract/inject trace context), then idle tracking
	handler := otelhttp.NewHandler(withIdleTracking(tracker, mux), "SandboxDaemon", otelhttp.WithServerName("sandbox-daemon"))

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("sandbox-daemon server error: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
