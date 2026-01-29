# Multi-stage build for sandbox daemon with Python runtime

FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /app/uno .

FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    SANDBOX_ROOT=/workspace \
    SANDBOX_PORT=8080

WORKDIR /workspace

# Common Python packages for data / scripting
#RUN pip install --no-cache-dir \
#    numpy \
#    pandas \
#    requests \
#    httpx \
#    pillow \
#    matplotlib \
#    scipy \
#    scikit-learn \
#    beautifulsoup4

# Basic CLI tools
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      bash \
      curl \
      wget \
      git \
      jq && \
    rm -rf /var/lib/apt/lists/*

# Copy backend binary
COPY --from=builder /app/uno /app/uno

EXPOSE 8080

ENTRYPOINT ["/app/uno"]
CMD ["sandbox-daemon"]

