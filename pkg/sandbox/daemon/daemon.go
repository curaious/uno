package daemon

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	defaultPort       = "8080"
	defaultSandboxDir = "/sandbox/workspace"
)

func NewSandboxDaemon() {
	port := getenv("SANDBOX_PORT", defaultPort)
	root := getenv("SANDBOX_ROOT", defaultSandboxDir)

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

	addr := ":" + port
	log.Printf("sandbox-daemon listening on %s (root=%s)", addr, filepath.Clean(root))

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("sandbox-daemon server error: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
