# =============================================================================
# Stage 1: Build Frontend
# =============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files
COPY frontend/package.json frontend/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY frontend/ .

# Build the app
RUN npm run build

# =============================================================================
# Stage 2: Build Backend
# =============================================================================
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/uno .

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
EXPOSE 8080 3000

ENTRYPOINT ["/app/entrypoint.sh"]

