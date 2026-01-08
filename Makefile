# Edge Logs Makefile
# Build automation for edge-logs project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=edge-logs-apiserver
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION ?= v0.1.0
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Docker
REGISTRY ?= ghcr.io/outpostos
IMAGE_NAME=$(REGISTRY)/edge-logs
TAG ?= $(VERSION)

.PHONY: all build clean test coverage lint help

all: test build

## Build
build: ## Build the binary file
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./cmd/apiserver

build-linux: ## Build the binary file for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v ./cmd/apiserver

## Test
test: ## Run tests
	$(GOTEST) -v ./...

test-race: ## Run tests with race detector
	$(GOTEST) -race -short ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Quality
lint: ## Run linter
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, install it from https://golangci-lint.run/usage/install/"; \
	fi

vet: ## Run go vet
	$(GOCMD) vet ./...

fmt: ## Run go fmt
	$(GOCMD) fmt ./...

## Dependencies
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

tidy: ## Clean up dependencies
	$(GOMOD) tidy

## Docker
docker-build: ## Build docker image
	docker build -t $(IMAGE_NAME):$(TAG) -f deploy/apiserver/Dockerfile .

docker-push: ## Push docker image
	docker push $(IMAGE_NAME):$(TAG)

docker-run: ## Run docker container
	docker run --rm -p 8080:8080 $(IMAGE_NAME):$(TAG)

## Development
run: ## Run the application
	$(GOCMD) run ./cmd/apiserver

dev: ## Run in development mode with auto-reload
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not found, running with 'go run' instead"; \
		$(GOCMD) run ./cmd/apiserver; \
	fi

## Maintenance
clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out
	rm -f coverage.html

install: ## Install the binary
	$(GOCMD) install $(LDFLAGS) ./cmd/apiserver

## Database
db-setup: ## Setup ClickHouse database (requires docker)
	@echo "Setting up ClickHouse database..."
	docker run -d --name clickhouse-server -p 9000:9000 -p 8123:8123 clickhouse/clickhouse-server:latest
	@echo "ClickHouse is starting up... wait a few seconds then run 'make db-init'"

db-init: ## Initialize database schema
	@echo "Initializing database schema..."
	@echo "Run: cat sqlscripts/clickhouse/*.sql | docker exec -i clickhouse-server clickhouse-client"

db-stop: ## Stop ClickHouse container
	docker stop clickhouse-server
	docker rm clickhouse-server

## Help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)