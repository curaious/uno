package sandbox

import "fmt"

// NotFoundError is returned when a sandbox for a given session does not exist.
type NotFoundError struct {
	SessionID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("sandbox not found for session %s", e.SessionID)
}
