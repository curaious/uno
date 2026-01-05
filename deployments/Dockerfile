# =============================================================================
# Stage 1: Build Frontend
# =============================================================================
# Use native platform for build - frontend output is platform-independent
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

# Enable pnpm via corepack (official way)
RUN corepack enable

WORKDIR /app

# Copy workspace files FIRST (important for caching)
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./

# Copy package.json files of all workspaces (no source yet)
COPY frontend/package.json frontend/package.json
COPY packages/uno-converse/package.json packages/uno-converse/package.json

# Install dependencies (workspace-aware)
RUN pnpm install --frozen-lockfile

# Now copy the full source
COPY . .

# Build UI package first (required)
RUN pnpm --filter @praveen001/uno-converse build

# Build frontend
RUN pnpm --filter frontend build

# =============================================================================
# Stage 2: Build Backend
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /app/uno .

# =============================================================================
# Stage 3: Final Runtime Image
# =============================================================================
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata nginx

# Create non-root user
RUN addgroup -g 1000 uno && \
    adduser -u 1000 -G uno -s /bin/sh -D uno

# Copy backend binary
COPY --from=backend-builder /app/uno /app/uno

# Copy frontend build
COPY --from=frontend-builder /app/frontend/build /app/static

# Copy nginx config
COPY deployments/nginx.conf /etc/nginx/http.d/default.conf

# Copy entrypoint script
COPY deployments/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Setup nginx directories
RUN chown -R uno:uno /app && \
    chown -R uno:uno /var/lib/nginx && \
    chown -R uno:uno /var/log/nginx && \
    mkdir -p /run/nginx && \
    chown -R uno:uno /run/nginx

USER uno

# Backend port and frontend port
EXPOSE 6060 3000

ENTRYPOINT ["/app/entrypoint.sh"]

