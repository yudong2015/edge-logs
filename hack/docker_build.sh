#!/bin/bash

# Docker build script for edge-logs
# This script builds the Docker image with proper tags and build args

set -e

# Default values
REGISTRY=${REGISTRY:-"ghcr.io/outpostos"}
IMAGE_NAME=${IMAGE_NAME:-"edge-logs"}
TAG=${TAG:-"latest"}
DOCKERFILE=${DOCKERFILE:-"deploy/apiserver/Dockerfile"}

# Build information
BUILD_DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")

# Full image name
FULL_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:${TAG}"

echo "Building Docker image..."
echo "Image: ${FULL_IMAGE_NAME}"
echo "Dockerfile: ${DOCKERFILE}"
echo "Build Date: ${BUILD_DATE}"
echo "Git Commit: ${GIT_COMMIT}"
echo "Git Tag: ${GIT_TAG}"
echo

# Build the image
docker build \
  --build-arg BUILD_DATE="${BUILD_DATE}" \
  --build-arg GIT_COMMIT="${GIT_COMMIT}" \
  --build-arg VERSION="${GIT_TAG}" \
  -f "${DOCKERFILE}" \
  -t "${FULL_IMAGE_NAME}" \
  .

echo
echo "✅ Image built successfully: ${FULL_IMAGE_NAME}"
echo

# Display image info
docker images "${REGISTRY}/${IMAGE_NAME}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

echo
echo "To run the image:"
echo "  docker run --rm -p 8080:8080 ${FULL_IMAGE_NAME}"
echo
echo "To push the image:"
echo "  docker push ${FULL_IMAGE_NAME}"