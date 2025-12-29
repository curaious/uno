package agents

import (
	"sync"
)

// agentRegistry is a global thread-safe registry for agent configurations.
// It allows Restate workflows to look up agent configs by name.
var (
	agentRegistry     = make(map[string]*Agent)
	agentRegistryLock sync.RWMutex
)

// RegisterAgent registers an agent configuration in the global registry.
// This is called when an agent is created with a Runtime that needs
// to look up the agent inside a workflow (e.g., RestateRuntime).
func RegisterAgent(name string, agent *Agent) {
	agentRegistryLock.Lock()
	defer agentRegistryLock.Unlock()
	agentRegistry[name] = agent
}

// GetAgent retrieves an agent configuration from the global registry.
// Returns nil if the agent is not found.
func GetAgent(name string) *Agent {
	agentRegistryLock.RLock()
	defer agentRegistryLock.RUnlock()
	return agentRegistry[name]
}

// UnregisterAgent removes an agent from the registry.
// Useful for cleanup or testing.
func UnregisterAgent(name string) {
	agentRegistryLock.Lock()
	defer agentRegistryLock.Unlock()
	delete(agentRegistry, name)
}
