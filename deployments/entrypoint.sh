#!/bin/sh
set -e

# Start nginx in background for frontend
nginx -g "daemon off;" &

# Start backend
exec /app/uno agent-server

