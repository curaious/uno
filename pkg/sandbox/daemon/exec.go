package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultTimeoutSeconds = 60
)

type ExecRequest struct {
	Command        string            `json:"command,omitempty"`         // for bash
	Args           []string          `json:"args,omitempty"`            // for bash
	Script         string            `json:"script,omitempty"`          // for python
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"` // defaults to 60
	Workdir        string            `json:"workdir,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
}

type ExecResponse struct {
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	ExitCode      int    `json:"exit_code"`
	DurationMilli int64  `json:"duration_ms"`
}

type sandboxHandler func(w http.ResponseWriter, r *http.Request, root string)

// withSandboxRoot injects the sandbox root into handlers.
func withSandboxRoot(root string, h sandboxHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, root)
	})
}

// withJSON sets JSON headers and common error handling.
func withJSON(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	})
}

func handleExecBash(w http.ResponseWriter, r *http.Request, root string) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid json: %v"}`, err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Command) == "" {
		http.Error(w, `{"error":"command is required"}`, http.StatusBadRequest)
		return
	}

	timeout := time.Duration(req.TimeoutSeconds)
	if timeout <= 0 {
		timeout = defaultTimeoutSeconds
	}
	timeout = timeout * time.Second

	workdir, err := resolvePath(root, req.Workdir)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	start := time.Now()

	// Execute through shell to support shell syntax (quotes, pipes, redirections, etc.)
	// If Args are provided, combine them with the command; otherwise use command as-is
	var shellCmd string
	if len(req.Args) > 0 {
		// Build command with args: "command arg1 arg2 ..."
		allArgs := append([]string{req.Command}, req.Args...)
		shellCmd = strings.Join(allArgs, " ")
	} else {
		// Use command as-is (may contain shell syntax)
		shellCmd = req.Command
	}

	// Use /bin/sh which is available in virtually all containers (POSIX-compliant)
	// This will work for most shell commands including date with format strings
	res, err := runCommand(ctx, "/bin/sh", []string{"-c", shellCmd}, workdir, req.Env)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Printf("bash exec error: %v", err)
	}
	res.DurationMilli = time.Since(start).Milliseconds()

	status := http.StatusOK
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
	}

	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(res)
}

func handleExecPython(w http.ResponseWriter, r *http.Request, root string) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid json: %v"}`, err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Script) == "" {
		http.Error(w, `{"error":"script is required"}`, http.StatusBadRequest)
		return
	}

	timeout := time.Duration(req.TimeoutSeconds)
	if timeout <= 0 {
		timeout = defaultTimeoutSeconds
	}
	timeout = timeout * time.Second

	workdir, err := resolvePath(root, req.Workdir)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(workdir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to create workdir: %v"}`, err), http.StatusInternalServerError)
		return
	}

	tmpFile, err := os.CreateTemp(workdir, "script-*.py")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to create temp file: %v"}`, err), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(req.Script); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to write script: %v"}`, err), http.StatusInternalServerError)
		return
	}

	if err := tmpFile.Sync(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to flush script: %v"}`, err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	start := time.Now()
	res, err := runCommand(ctx, "python3", []string{tmpFile.Name()}, workdir, req.Env)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Printf("python exec error: %v", err)
	}
	res.DurationMilli = time.Since(start).Milliseconds()

	status := http.StatusOK
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
	}

	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(res)
}

func runCommand(ctx context.Context, name string, args []string, workdir string, env map[string]string) (*ExecResponse, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	slog.InfoContext(ctx, "working directory:"+workdir)

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	slog.InfoContext(ctx, "Executing command: "+cmd.String())
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutPipe)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(&stderrBuf, stderrPipe)
		done <- struct{}{}
	}()

	// Wait for both copy goroutines
	<-done
	<-done

	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) {
			// distinguish timeout
			return &ExecResponse{
				Stdout:   stdoutBuf.String(),
				Stderr:   stderrBuf.String(),
				ExitCode: -1,
			}, err
		} else {
			return nil, fmt.Errorf("wait: %w", err)
		}
	}

	return &ExecResponse{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
	}, err
}

// resolvePath returns an absolute path inside the sandbox root.
// If rel is empty, root is returned.
func resolvePath(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return root, nil
	}
	cleanRoot := filepath.Clean(root)
	target := filepath.Join(cleanRoot, rel)
	target = filepath.Clean(target)

	if !strings.HasPrefix(target, cleanRoot) {
		return "", fmt.Errorf("path escapes sandbox root")
	}
	return target, nil
}
