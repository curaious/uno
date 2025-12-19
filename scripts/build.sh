#!/bin/bash
set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Get the repo root (parent of scripts directory)
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
IMAGE_NAME="${IMAGE_NAME:-ghcr.io/praveen001/uno}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-false}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --tag|-t)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --push|-p)
            PUSH="true"
            shift
            ;;
        --platform)
            PLATFORMS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "Building image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo "Platforms: ${PLATFORMS}"
echo "Push: ${PUSH}"
echo ""

cd "${REPO_ROOT}"

if [ "${PUSH}" = "true" ]; then
    docker buildx build \
        --platform "${PLATFORMS}" \
        --file deployments/Dockerfile \
        --tag "${IMAGE_NAME}:${IMAGE_TAG}" \
        --push \
        .
else
    docker buildx build \
        --platform linux/amd64 \
        --file deployments/Dockerfile \
        --tag "${IMAGE_NAME}:${IMAGE_TAG}" \
        --load \
        .
fi

echo ""
echo "✅ Build complete: ${IMAGE_NAME}:${IMAGE_TAG}"

