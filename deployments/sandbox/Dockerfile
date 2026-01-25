# Multi-stage build for sandbox daemon with Python runtime

FROM golang:1.22-bullseye AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /src/cmd/sandbox-daemon
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sandbox-daemon .

FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    SANDBOX_ROOT=/workspace \
    SANDBOX_PORT=8080

WORKDIR /workspace

# Common Python packages for data / scripting
RUN pip install --no-cache-dir \
    numpy \
    pandas \
    requests \
    httpx \
    pillow \
    matplotlib \
    scipy \
    scikit-learn \
    beautifulsoup4

# Basic CLI tools
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      bash \
      curl \
      wget \
      git \
      jq && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/sandbox-daemon /usr/local/bin/sandbox-daemon

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/sandbox-daemon"]

