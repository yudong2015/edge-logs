#!/bin/bash
set -e

# Rollback deployment
ENV="${1:-dev}"
COMPONENT="${2:-all}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../envs/${ENV}.env"

REMOTE_HOST="hw101"

rollback_component() {
    local comp="$1"
    echo "Rolling back $comp in $NAMESPACE..."

    if ssh "${REMOTE_HOST}" "kubectl rollout undo deployment/$comp -n $NAMESPACE"; then
        echo "✓ Rollback initiated for $comp"
        ssh "${REMOTE_HOST}" "kubectl rollout status deployment/$comp -n $NAMESPACE --timeout=300s"
        echo "✓ Rollback completed for $comp"
    else
        echo "✗ Failed to rollback $comp"
        return 1
    fi
}

case "$COMPONENT" in
    apiserver|api)
        rollback_component "edge-logs-apiserver"
        ;;
    frontend|fe)
        rollback_component "edge-logs-frontend"
        ;;
    all)
        rollback_component "edge-logs-apiserver"
        rollback_component "edge-logs-frontend"
        ;;
    *)
        echo "Usage: $0 [env] [component]"
        echo "Components: apiserver, frontend, all"
        exit 1
        ;;
esac

echo "Rollback completed for $ENV environment"