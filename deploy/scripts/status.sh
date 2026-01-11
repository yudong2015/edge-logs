#!/bin/bash
set -e

# Check deployment status
ENV="${1:-dev}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../envs/${ENV}.env"

REMOTE_HOST="hw101"

echo "=== Edge Logs Status ($ENV) ==="
echo "Namespace: $NAMESPACE"
echo ""

echo "=== Pods ==="
ssh "${REMOTE_HOST}" "kubectl get pods -n $NAMESPACE -o wide"
echo ""

echo "=== Services ==="
ssh "${REMOTE_HOST}" "kubectl get svc -n $NAMESPACE"
echo ""

echo "=== Recent Events ==="
ssh "${REMOTE_HOST}" "kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -10"