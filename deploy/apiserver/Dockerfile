# Multi-stage build for optimal image size
# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set up build environment
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy go mod and sum files for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
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