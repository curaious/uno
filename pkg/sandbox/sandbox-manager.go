package sandbox

// Package sandbox provides primitives for managing per-session sandbox pods.
//
// This file intentionally only defines interfaces that higher-level code can depend on.
// The concrete Kubernetes-backed implementation lives in manager.go.

import "context"

// Manager defines the lifecycle operations for sandboxes.
type Manager interface {
	// CreateSandbox ensures a sandbox exists for the given session, creating one if needed.
	CreateSandbox(ctx context.Context, image string, agentName string, namespace string, sessionID string) (*SandboxHandle, error)
	// GetSandbox returns the handle for an existing sandbox.
	GetSandbox(ctx context.Context, sessionID string) (*SandboxHandle, error)
	// DeleteSandbox tears down the sandbox for the given session.
	DeleteSandbox(ctx context.Context, sessionID string) error
}

// SandboxHandle represents a running sandbox pod bound to a session.
type SandboxHandle struct {
	SessionID string
	PodName   string
	PodIP     string
	Port      int
}
