package k8s_sandbox

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/curaious/uno/pkg/sandbox"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config defines how sandboxes (pods) are created.
type Config struct {
	// Namespace where sandbox pods will be created.
	Namespace string

	// RootDir is the base directory path for workspace volumes.
	RootDir string

	// Resource hints (K8s quantities, e.g. "500m", "1Gi").
	CPU    string
	Memory string

	// Port the sandbox daemon listens on inside the pod.
	Port int

	// Storage configuration for PersistentVolumeClaims
	StorageClass string
	StorageSize  string // e.g., "10Gi"

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

	if cfg.StorageSize == "" {
		cfg.StorageSize = "50Mi"
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

func (m *kubeManager) CreateSandbox(ctx context.Context, image string, agentName string, namespace string, sessionID string) (*sandbox.SandboxHandle, error) {
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

	// Ensure session workspace PVC exists (only PVC managed by sandbox)
	workspacePVCName, err := m.ensureSessionPVC(ctx, agentName, namespace, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ensure session PVC: %w", err)
	}

	// Shared workspace PVC name (managed by agent orchestrator)
	sharedWorkspacePVCName := fmt.Sprintf("pvc-%s-workspace", agentName)

	// Create volumes and volume mounts
	volumes := []corev1.Volume{
		{
			Name: "shared-workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: sharedWorkspacePVCName,
					ReadOnly:  true,
				},
			},
		},
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: workspacePVCName,
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "shared-workspace",
			MountPath: "/sandbox/global-workspace",
			SubPath:   "workspace",
			ReadOnly:  true,
		},
		{
			Name:      "shared-workspace",
			MountPath: "/sandbox/named-workspace",
			SubPath:   path.Join("namespaces", namespace, "workspace"),
			ReadOnly:  true,
		},
		{
			Name:      "workspace",
			MountPath: "/sandbox/workspace",
		},
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
			Volumes:       volumes,
			Containers: []corev1.Container{
				{
					Name:         "sandbox",
					Image:        image,
					WorkingDir:   "/sandbox",
					VolumeMounts: volumeMounts,
					Env: []corev1.EnvVar{
						{Name: "SANDBOX_ROOT", Value: "/sandbox/workspace"},
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

// ensureSessionPVC creates or gets the session-specific workspace PVC.
// The shared workspace PVC (for global and named workspaces) is managed by the agent orchestrator
// and should already exist. Returns the PVC name for the session workspace.
func (m *kubeManager) ensureSessionPVC(ctx context.Context, agentName string, namespace string, sessionID string) (string, error) {
	// Session workspace PVC (unique per session, read-write)
	workspacePVCName := fmt.Sprintf("pvc-%s-%s-%s-workspace", agentName, namespace, sessionID)
	if err := m.createOrGetPVC(ctx, workspacePVCName, agentName, path.Join("namespaces", namespace, "sessions", sessionID)); err != nil {
		return "", fmt.Errorf("workspace PVC: %w", err)
	}

	return workspacePVCName, nil
}

// createOrGetPVC creates a PVC if it doesn't exist, or returns the existing one.
func (m *kubeManager) createOrGetPVC(ctx context.Context, pvcName string, agentName string, subPath string) error {
	// Check if PVC already exists
	_, err := m.client.CoreV1().PersistentVolumeClaims(m.cfg.Namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		// PVC exists, return success
		return nil
	}

	// Create new PVC
	storageQuantity, err := resource.ParseQuantity(m.cfg.StorageSize)
	if err != nil {
		return fmt.Errorf("invalid storage size %s: %w", m.cfg.StorageSize, err)
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: m.cfg.Namespace,
			Labels: map[string]string{
				"app":       "uno-sandbox",
				"agent":     agentName,
				"managed":   "uno",
				"component": "workspace",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageQuantity,
				},
			},
		},
	}

	if m.cfg.StorageClass != "" {
		pvc.Spec.StorageClassName = &m.cfg.StorageClass
	}

	_, err = m.client.CoreV1().PersistentVolumeClaims(m.cfg.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create PVC %s: %w", pvcName, err)
	}

	return nil
}
