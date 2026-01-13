# Multi-stage build for optimal image size
# Stage 1: Build the Go binary
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

# Build arguments
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Set up build environment
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy go mod and sum files for dependency caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-w -s -extldflags '-static' -X main.Version=$VERSION -X main.BuildDate=$BUILD_DATE -X main.GitCommit=$GIT_COMMIT -X github.com/outpostos/edge-logs/pkg/apiserver.version=$VERSION" \
    -a -installsuffix cgo \
    -o edge-logs-apiserver \
    ./cmd/apiserver

# Stage 2: Create final minimal image
FROM scratch

# Import ca-certificates from builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /src/edge-logs-apiserver /edge-logs-apiserver

# Copy default configuration
COPY --from=builder /src/config/config.yaml /config/config.yaml

# Expose port
EXPOSE 8080

# Set up non-root user (using numeric IDs for scratch image)
USER 1000:1000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/edge-logs-apiserver", "--help"] || exit 1

# Run the binary
ENTRYPOINT ["/edge-logs-apiserver"]
