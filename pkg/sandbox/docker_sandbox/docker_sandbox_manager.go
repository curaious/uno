package docker_sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/curaious/uno/pkg/sandbox"
)

type Config struct {
	Network string

	// Port the sandbox daemon listens on inside the pod.
	Port int
}

type DockerSandboxManager struct {
	cfg Config

	mu        sync.RWMutex
	bySession map[string]*sandbox.SandboxHandle
}

func NewManager(cfg Config) *DockerSandboxManager {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	return &DockerSandboxManager{
		cfg:       cfg,
		bySession: make(map[string]*sandbox.SandboxHandle),
	}
}

func (m *DockerSandboxManager) CreateSandbox(ctx context.Context, image string, sessionID string) (*sandbox.SandboxHandle, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}

	// Fast path: cached
	if h := m.getCached(sessionID); h != nil {
		return h, nil
	}

	name := fmt.Sprintf("sandbox-%s", sessionID)

	// If container already exists, reuse it
	if ip, err := inspectContainerIP(ctx, name); err == nil && ip != "" {
		h := &sandbox.SandboxHandle{
			SessionID: sessionID,
			PodName:   name,
			PodIP:     ip,
			Port:      m.cfg.Port,
		}
		m.setCached(h)
		return h, nil
	}

	// Build docker run args
	args := []string{"run", "-d", "--name", name}
	if m.cfg.Network != "" {
		args = append(args, "--network", m.cfg.Network)
	}
	// optional envs
	args = append(args,
		"-e", fmt.Sprintf("SANDBOX_PORT=%d", m.cfg.Port),
		"-e", "SANDBOX_ROOT=/workspace",
		image,
	)

	if err := runDocker(ctx, args...); err != nil {
		return nil, fmt.Errorf("docker run: %w", err)
	}

	// Wait for container IP
	var ip string
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		ip, _ = inspectContainerIP(ctx, name)
		if ip != "" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if ip == "" {
		return nil, fmt.Errorf("container %s did not getCached an IP", name)
	}

	h := &sandbox.SandboxHandle{
		SessionID: sessionID,
		PodName:   name,
		PodIP:     ip,
		Port:      m.cfg.Port,
	}
	m.setCached(h)

	return h, nil
}

func (m *DockerSandboxManager) GetSandbox(_ context.Context, sessionID string) (*sandbox.SandboxHandle, error) {
	if h := m.getCached(sessionID); h != nil {
		return h, nil
	}
	return nil, fmt.Errorf("sandbox not found for session %s", sessionID)
}

func (m *DockerSandboxManager) DeleteSandbox(ctx context.Context, sessionID string) error {
	h := m.getCached(sessionID)
	if h == nil {
		return nil
	}

	args := []string{"rm", "-f", h.PodName}
	if err := runDocker(ctx, args...); err != nil {
		return fmt.Errorf("docker rm: %w", err)
	}

	m.mu.Lock()
	delete(m.bySession, sessionID)
	m.mu.Unlock()
	return nil
}

// --- helpers ---

func (m *DockerSandboxManager) getCached(sessionID string) *sandbox.SandboxHandle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bySession[sessionID]
}

func (m *DockerSandboxManager) setCached(h *sandbox.SandboxHandle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bySession[h.SessionID] = h
}

func runDocker(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func inspectContainerIP(ctx context.Context, name string) (string, error) {
	// Grab container IP from default network
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	ip := strings.TrimSpace(out.String())
	return ip, nil
}
