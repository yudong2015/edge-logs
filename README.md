# Edge Logs

Edge Logs is a high-performance log aggregation and query system designed for Kubernetes edge environments. It provides fast log search, filtering, and analysis capabilities with ClickHouse as the storage backend.

## Overview

Edge Logs aggregates logs from Kubernetes edge clusters and provides:
- **Fast Log Search**: Millisecond-level query response times
- **Dataset-based Isolation**: Multi-tenant log access with dataset-based routing
- **Rich Filtering**: Time range, namespace, pod, and content-based filtering
- **K8s Integration**: Automatic metadata enrichment from Kubernetes API
- **RESTful API**: go-restful based API for log queries
- **Web Interface**: Professional web UI for log exploration

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web UI        │    │   API Server    │    │   ClickHouse    │
│   (React)       │───▶│   (go-restful) │───▶│   Database      │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Kubernetes     │
                       │  API Server     │
                       │  (metadata)     │
                       └─────────────────┘
```

## Prerequisites

- **Go 1.23+**: For building and development
- **ClickHouse**: Database for log storage
- **Kubernetes**: Source of logs and metadata (optional for development)

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/outpostos/edge-logs.git
cd edge-logs

# Download dependencies
make deps

# Build the binary
make build

# Or run directly
make run
```

### 2. Setup ClickHouse (Development)

```bash
# Start ClickHouse with Docker
make db-setup

# Wait a few seconds, then initialize schema
make db-init
```

### 3. Configure

Copy and modify the configuration:

```bash
cp config/config.yaml config/local.yaml
# Edit config/local.yaml with your settings

# Run with custom config
CONFIG_FILE=config/local.yaml make run
```

### 4. Test the API

```bash
# Health check
curl http://localhost:8080/api/v1alpha1/health

# Example response:
{
  "status": "healthy",
  "service": "edge-logs-apiserver",
  "version": "v0.1.0",
  "timestamp": "2024-01-09T00:00:00Z"
}
```

## Configuration

### Environment Variables

Key configuration options can be overridden with environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `CONFIG_FILE` | Path to configuration file | `config/config.yaml` |
| `SERVER_HOST` | HTTP server bind address | `0.0.0.0` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `CLICKHOUSE_HOST` | ClickHouse server host | `localhost` |
| `CLICKHOUSE_USERNAME` | ClickHouse username | `default` |
| `CLICKHOUSE_PASSWORD` | ClickHouse password | (empty) |

### Configuration File

See [`config/config.yaml`](config/config.yaml) for full configuration options including:
- Server settings (host, port, timeouts)
- ClickHouse connection parameters
- Kubernetes client configuration
- Logging preferences

## Development

### Make Targets

```bash
# Build and test
make all              # Run tests and build
make build           # Build binary
make test            # Run tests
make test-coverage   # Run tests with coverage

# Code quality
make lint            # Run golangci-lint
make vet             # Run go vet
make fmt             # Format code

# Development workflow
make run             # Run locally
make dev             # Run with auto-reload (requires air)

# Docker
make docker-build    # Build container image
make docker-run      # Run in container

# Database
make db-setup        # Start ClickHouse container
make db-init         # Initialize schema
make db-stop         # Stop database

# Maintenance
make clean           # Remove build artifacts
make deps            # Download dependencies
make tidy            # Clean up go.mod
```

### Project Structure

```
edge-logs/
├── cmd/apiserver/           # Application entry point
├── pkg/                     # Library code
│   ├── apiserver/          # HTTP server setup
│   ├── oapis/              # API handlers
│   ├── model/              # Data models
│   ├── repository/         # Data access layer
│   ├── service/            # Business logic
│   ├── middleware/         # HTTP middleware
│   └── config/             # Configuration management
├── config/                  # Configuration files
├── deploy/                  # Deployment manifests
├── test/                    # Test files
├── sqlscripts/             # Database schema
└── hack/                   # Build scripts
```

### Testing

```bash
# Run all tests
make test

# Run with race detection
make test-race

# Generate coverage report
make test-coverage
open coverage.html
```

### Adding Dependencies

```bash
# Add new dependency
go get github.com/example/package

# Clean up
make tidy
```

## API Documentation

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1alpha1/health` | Health check |
| `GET` | `/api/v1alpha1/logs` | Query logs (coming soon) |
| `GET` | `/api/v1alpha1/datasets` | List datasets (coming soon) |

### Health Check

```bash
GET /api/v1alpha1/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "edge-logs-apiserver",
  "version": "v0.1.0",
  "timestamp": "2024-01-09T00:00:00Z"
}
```

## Deployment

### Docker

```bash
# Build image
make docker-build

# Run container
docker run -p 8080:8080 \
  -e CLICKHOUSE_HOST=your-clickhouse-host \
  ghcr.io/outpostos/edge-logs:v0.1.0
```

### Kubernetes

```bash
# Apply manifests (coming soon)
kubectl apply -f deploy/
```

## Contributing

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature-name`
3. **Make** your changes
4. **Test** thoroughly: `make test lint`
5. **Commit** with clear messages
6. **Push** to your fork: `git push origin feature-name`
7. **Submit** a pull request

### Code Standards

- **Go Code**: Follow standard Go conventions
- **Testing**: Maintain >80% code coverage
- **Documentation**: Update README and code comments
- **Linting**: All code must pass `make lint`

### Development Workflow

```bash
# Setup development environment
git clone https://github.com/outpostos/edge-logs.git
cd edge-logs
make deps

# Make changes and test
make test lint

# Run locally
make run
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Documentation**: [Wiki](https://github.com/outpostos/edge-logs/wiki)

---

**Built with ❤️ for Kubernetes edge computing**