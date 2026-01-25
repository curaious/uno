package k8s_sandbox

import "time"

// ExecResult is the normalized result of a command execution in the sandbox.
type ExecResult struct {
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	ExitCode      int    `json:"exit_code"`
	DurationMilli int64  `json:"duration_ms"`
}
