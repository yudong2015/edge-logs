#!/bin/bash
set -e

# View logs for edge-logs components
ENV="${1:-dev}"
COMPONENT="${2:-apiserver}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../envs/${ENV}.env"

REMOTE_HOST="hw101"

case "$COMPONENT" in
    apiserver|api)
        echo "=== APIServer Logs ==="
        ssh "${REMOTE_HOST}" "kubectl logs -n $NAMESPACE -l app=edge-logs-apiserver -f --tail=100"
        ;;
    frontend|fe)
        echo "=== Frontend Logs ==="
        ssh "${REMOTE_HOST}" "kubectl logs -n $NAMESPACE -l app=edge-logs-frontend -f --tail=100"
        ;;
    clickhouse|ch)
        echo "=== ClickHouse Logs ==="
        ssh "${REMOTE_HOST}" "kubectl logs -n $NAMESPACE -l app=clickhouse -f --tail=100"
        ;;
    all)
        echo "=== All Logs ==="
        ssh "${REMOTE_HOST}" "kubectl logs -n $NAMESPACE --all-containers=true -f --tail=50"
        ;;
    *)
        echo "Usage: $0 [env] [component]"
        echo "Components: apiserver, frontend, clickhouse, all"
        exit 1
        ;;
esac