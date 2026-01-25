package k8s_sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/curaious/uno/pkg/sandbox"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config defines how sandboxes (pods) are created.
type Config struct {
	// Namespace where sandbox pods will be created.
	Namespace string

	// Resource hints (K8s quantities, e.g. "500m", "1Gi").
	CPU    string
	Memory string

	// Port the sandbox daemon listens on inside the pod.
	Port int

	// Optional TTL controls how long to keep idle sandboxes.
	TTL time.Duration
}

// kubeManager is a Kubernetes-backed implementation of Manager.
type kubeManager struct {
	client *kubernetes.Clientset
	cfg    Config

	mu        sync.RWMutex
	bySession map[string]*sandbox.SandboxHandle
}

// NewManager creates a sandbox manager that uses the in-cluster
// configuration by default, falling back to the default rest config.
func NewManager(cfg Config) (sandbox.Manager, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "uno-sandbox"
	}

	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		restCfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &kubeManager{
		client:    client,
		cfg:       cfg,
		bySession: make(map[string]*sandbox.SandboxHandle),
	}, nil
}

func (m *kubeManager) CreateSandbox(ctx context.Context, image string, sessionID string) (*sandbox.SandboxHandle, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}

	// Fast path: already have a handle in memory.
	if h := m.getCached(sessionID); h != nil {
		return h, nil
	}

	podName := fmt.Sprintf("sandbox-%s", sessionID)

	// Check if pod already exists.
	pod, err := m.client.CoreV1().Pods(m.cfg.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err == nil && pod != nil {
		handle := &sandbox.SandboxHandle{
			SessionID: sessionID,
			PodName:   pod.Name,
			PodIP:     pod.Status.PodIP,
			Port:      m.cfg.Port,
		}
		m.setCached(handle)
		return handle, nil
	}

	// Create new pod
	podSpec := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: m.cfg.Namespace,
			Labels: map[string]string{
				"app":      "uno-sandbox",
				"session":  sessionID,
				"managed":  "uno",
				"provider": "sandbox-daemon",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "sandbox",
					Image: image,
					Env: []corev1.EnvVar{
						{Name: "SANDBOX_ROOT", Value: "/workspace"},
						{Name: "SANDBOX_PORT", Value: fmt.Sprintf("%d", m.cfg.Port)},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: int32(m.cfg.Port),
						},
					},
					Resources: corev1.ResourceRequirements{},
				},
			},
		},
	}

	_, err = m.client.CoreV1().Pods(m.cfg.Namespace).Create(ctx, podSpec, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create sandbox pod: %w", err)
	}

	// Wait for pod to be running and have an IP.
	pod, err = m.waitForRunning(ctx, podName)
	if err != nil {
		return nil, err
	}

	handle := &sandbox.SandboxHandle{
		SessionID: sessionID,
		PodName:   pod.Name,
		PodIP:     pod.Status.PodIP,
		Port:      m.cfg.Port,
	}
	m.setCached(handle)

	return handle, nil
}

func (m *kubeManager) GetSandbox(ctx context.Context, sessionID string) (*sandbox.SandboxHandle, error) {
	if h := m.getCached(sessionID); h != nil {
		return h, nil
	}
	return nil, &sandbox.NotFoundError{SessionID: sessionID}
}

func (m *kubeManager) DeleteSandbox(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	handle, ok := m.bySession[sessionID]
	if ok {
		delete(m.bySession, sessionID)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}

	propagation := metav1.DeletePropagationBackground
	if err := m.client.CoreV1().Pods(m.cfg.Namespace).Delete(ctx, handle.PodName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}); err != nil {
		return fmt.Errorf("delete sandbox pod: %w", err)
	}
	return nil
}

func (m *kubeManager) waitForRunning(ctx context.Context, podName string) (*corev1.Pod, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for pod %s: %w", podName, ctx.Err())
		case <-timeout:
			return nil, fmt.Errorf("timed out waiting for pod %s to become running", podName)
		case <-ticker.C:
			pod, err := m.client.CoreV1().Pods(m.cfg.Namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if pod.Status.Phase == corev1.PodRunning && pod.Status.PodIP != "" {
				return pod, nil
			}
		}
	}
}

func (m *kubeManager) getCached(sessionID string) *sandbox.SandboxHandle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bySession[sessionID]
}

func (m *kubeManager) setCached(h *sandbox.SandboxHandle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bySession[h.SessionID] = h
}
