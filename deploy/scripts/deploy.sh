#!/bin/bash
set -e

# Edge Logs K8s Deployment Script
# Usage: ./deploy.sh [env] [image_tag]
# Environment: dev, staging, prod (default: dev)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEPLOY_DIR="${PROJECT_ROOT}/deploy"

# Default values
ENV="${1:-dev}"
IMAGE_TAG="${2:-${GITHUB_SHA:0:7}}"
REMOTE_HOST="hw101"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Halt function
halt() {
    log_error "HALT: $1"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        halt "kubectl is not installed"
    fi

    # Check if envsubst is available
    if ! command -v envsubst &> /dev/null; then
        halt "envsubst is not installed (install gettext package)"
    fi

    log_success "Prerequisites check passed"
}

# Setup kubectl context based on environment
setup_kubectl_context() {
    if [[ -n "${CLUSTER_HOST:-}" ]]; then
        # Remote cluster via SSH
        log_info "Using remote cluster: $CLUSTER_HOST"
        KUBECTL_CMD="ssh $CLUSTER_HOST kubectl"
    elif [[ -n "${CLUSTER_CONTEXT:-}" ]]; then
        # Local cluster context
        log_info "Using local cluster context: $CLUSTER_CONTEXT"
        kubectl config use-context "$CLUSTER_CONTEXT" || halt "Failed to switch to context: $CLUSTER_CONTEXT"
        KUBECTL_CMD="kubectl"
    else
        # Use local kubectl by default
        log_info "Using local kubectl"
        KUBECTL_CMD="kubectl"
    fi

    # Test cluster connectivity
    if ! $KUBECTL_CMD get nodes &> /dev/null; then
        halt "集群连接失败"
    fi

    log_success "Cluster connectivity confirmed"
}

# Load environment configuration
load_env_config() {
    local env_file="${DEPLOY_DIR}/envs/${ENV}.env"

    if [[ ! -f "$env_file" ]]; then
        halt "Environment file not found: $env_file"
    fi

    log_info "Loading environment configuration: $env_file"
    source "$env_file"

    # Override IMAGE_TAG if provided
    if [[ -n "$IMAGE_TAG" && "$IMAGE_TAG" != "dev" ]]; then
        export IMAGE_TAG="$IMAGE_TAG"
    fi

    # Export required variables for envsubst
    export NAMESPACE
    export DOCKER_REGISTRY
    export DOCKER_REPO
    export IMAGE_TAG

    # Production deployment confirmation
    if [[ "$ENV" == "prod" ]]; then
        halt "生产部署需确认"
    fi

    log_info "Environment: $ENV"
    log_info "Namespace: $NAMESPACE"
    log_info "Image Tag: $IMAGE_TAG"
    log_info "Registry: $DOCKER_REGISTRY/$DOCKER_REPO"
}

# Apply manifests with variable substitution
apply_manifests() {
    log_info "Deploying to namespace: $NAMESPACE"

    # Create namespace if not exists
    $KUBECTL_CMD create namespace $NAMESPACE --dry-run=client -o yaml | $KUBECTL_CMD apply -f -

    # Apply manifests in order with variable substitution
    local manifests=(
        "00-namespace.yaml"
        "01-clickhouse.yaml"
        "02-apiserver.yaml"
        "03-frontend.yaml"
    )

    for manifest in "${manifests[@]}"; do
        local manifest_file="${DEPLOY_DIR}/k8s/${manifest}"

        if [[ ! -f "$manifest_file" ]]; then
            log_warn "Manifest not found: $manifest_file, skipping..."
            continue
        fi

        log_info "Applying manifest: $manifest"

        # Use envsubst for variable substitution and apply
        envsubst < "$manifest_file" | $KUBECTL_CMD apply -f - -n $NAMESPACE
    done
}

# Wait for deployments to be ready
wait_for_deployments() {
    log_info "Waiting for deployments to be ready..."

    local deployments=("clickhouse" "edge-logs-apiserver" "edge-logs-frontend")

    for deployment in "${deployments[@]}"; do
        log_info "Waiting for $deployment..."

        # Check if deployment exists
        if $KUBECTL_CMD get deployment $deployment -n $NAMESPACE &> /dev/null; then
            if ! $KUBECTL_CMD rollout status deployment/$deployment -n $NAMESPACE --timeout=300s; then
                halt "Pod 启动失败: $deployment"
            fi
        elif $KUBECTL_CMD get statefulset $deployment -n $NAMESPACE &> /dev/null; then
            if ! $KUBECTL_CMD rollout status statefulset/$deployment -n $NAMESPACE --timeout=300s; then
                halt "Pod 启动失败: $deployment"
            fi
        else
            log_warn "Deployment/StatefulSet $deployment not found, skipping wait..."
        fi
    done

    log_success "All deployments are ready"
}

# Verify deployment health
verify_deployment() {
    log_info "Verifying deployment health..."

    # Check APIServer health
    if $KUBECTL_CMD exec -n $NAMESPACE deployment/edge-logs-apiserver -- wget -qO- http://localhost:8080/api/v1alpha1/health &> /dev/null; then
        log_success "APIServer health check passed"
    else
        log_error "APIServer health check failed"
        return 1
    fi

    # Check Frontend health
    if $KUBECTL_CMD exec -n $NAMESPACE deployment/edge-logs-frontend -- curl -sf http://localhost/healthz &> /dev/null; then
        log_success "Frontend health check passed"
    else
        log_error "Frontend health check failed"
        return 1
    fi

    # Check ClickHouse health
    if $KUBECTL_CMD exec -n $NAMESPACE clickhouse-0 -- wget -qO- http://localhost:8123/ping &> /dev/null; then
        log_success "ClickHouse health check passed"
    else
        log_error "ClickHouse health check failed"
        return 1
    fi

    # Check service connectivity
    if $KUBECTL_CMD exec -n $NAMESPACE deployment/edge-logs-frontend -- curl -sf http://edge-logs-apiserver.$NAMESPACE.svc.cluster.local:8080/api/v1alpha1/health &> /dev/null; then
        log_success "Service connectivity check passed"
    else
        log_error "Service connectivity check failed"
        return 1
    fi

    log_success "All health checks passed"
}

# Show deployment status
show_status() {
    log_info "Deployment Status for $ENV environment:"
    echo ""

    # Show pods
    log_info "Pods:"
    $KUBECTL_CMD get pods -n $NAMESPACE -o wide
    echo ""

    # Show services
    log_info "Services:"
    $KUBECTL_CMD get svc -n $NAMESPACE
    echo ""

    # Show ingress if enabled
    if [[ "${INGRESS_ENABLED:-false}" == "true" ]]; then
        log_info "Ingress:"
        $KUBECTL_CMD get ingress -n $NAMESPACE || log_warn "No ingress found"
        echo ""
    fi
}

# Main execution
main() {
    log_info "Starting deployment for environment: $ENV"
    log_info "Image tag: $IMAGE_TAG"

    check_prerequisites
    load_env_config
    setup_kubectl_context
    apply_manifests
    wait_for_deployments

    if verify_deployment; then
        log_success "Deployment completed successfully!"
        show_status
    else
        halt "部署验证失败"
    fi
}

# Execute main function
main "$@"