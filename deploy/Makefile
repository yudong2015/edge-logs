# Edge Logs Deployment Makefile

.PHONY: help deploy status logs rollback clean

ENV ?= dev
COMPONENT ?= all
IMAGE_TAG ?= latest

help: ## Show this help message
	@echo 'Usage: make [target] [ENV=env] [COMPONENT=component] [IMAGE_TAG=tag]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''
	@echo 'Environments: dev (default), staging, prod'
	@echo 'Components: apiserver, frontend, clickhouse, all'

deploy: ## Deploy to specified environment
	@echo "Deploying to $(ENV) environment with tag $(IMAGE_TAG)..."
	./scripts/deploy.sh $(ENV) $(IMAGE_TAG)

status: ## Check deployment status
	@echo "Checking status for $(ENV) environment..."
	./scripts/status.sh $(ENV)

logs: ## View logs for specified component
	@echo "Showing logs for $(COMPONENT) in $(ENV) environment..."
	./scripts/logs.sh $(ENV) $(COMPONENT)

rollback: ## Rollback deployment
	@echo "Rolling back $(COMPONENT) in $(ENV) environment..."
	./scripts/rollback.sh $(ENV) $(COMPONENT)

clean: ## Clean up resources (requires confirmation)
	@echo "This will delete all resources in $(ENV) environment!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	kubectl delete namespace edge-logs-$(ENV) --ignore-not-found

# Shortcuts for common operations
dev: ENV=dev
dev: deploy

staging: ENV=staging
staging: deploy

prod: ENV=prod
prod: deploy

# Component-specific shortcuts
api-logs: COMPONENT=apiserver
api-logs: logs

fe-logs: COMPONENT=frontend
fe-logs: logs

ch-logs: COMPONENT=clickhouse
ch-logs: logs