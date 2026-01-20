#!/bin/sh
set -e

# Default command if none provided
COMMAND="${1:-agent-server}"

# Start nginx in background for frontend (only for agent-server)
if [ "$COMMAND" = "agent-server" ]; then
  nginx -g "daemon off;" &
fi

# Start backend with the provided command (or default)
exec /app/uno "$@"

