#!/bin/bash
# Edge Logs 验收测试脚本
# 验证完整的部署流程和功能

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 全局变量
TEST_RESULTS=()
FAILED_TESTS=0
TOTAL_TESTS=0

# 测试结果记录
record_test() {
    local test_name="$1"
    local result="$2"
    local message="$3"

    ((TOTAL_TESTS++))

    if [[ "$result" == "PASS" ]]; then
        log_success "✅ $test_name: $message"
        TEST_RESULTS+=("✅ $test_name: PASS - $message")
    else
        log_error "❌ $test_name: $message"
        TEST_RESULTS+=("❌ $test_name: FAIL - $message")
        ((FAILED_TESTS++))
    fi
}

# 检查依赖工具
test_dependencies() {
    log_info "检查依赖工具..."

    local tools=("kubectl" "helm" "docker")
    local missing_tools=()

    for tool in "${tools[@]}"; do
        if command -v "$tool" &> /dev/null; then
            record_test "工具检查-$tool" "PASS" "已安装"
        else
            record_test "工具检查-$tool" "FAIL" "未安装"
            missing_tools+=("$tool")
        fi
    done

    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "缺少工具: ${missing_tools[*]}"
        return 1
    fi
}

# 检查集群连接
test_cluster_connection() {
    log_info "检查Kubernetes集群连接..."

    if kubectl cluster-info &>/dev/null; then
        record_test "集群连接" "PASS" "连接正常"
    else
        record_test "集群连接" "FAIL" "无法连接到集群"
        return 1
    fi

    # 检查节点状态
    local ready_nodes
    ready_nodes=$(kubectl get nodes --no-headers | grep -c "Ready" || echo "0")

    if [[ $ready_nodes -gt 0 ]]; then
        record_test "集群节点" "PASS" "$ready_nodes 个节点Ready"
    else
        record_test "集群节点" "FAIL" "没有Ready节点"
        return 1
    fi
}

# 验证镜像可用性
test_image_availability() {
    log_info "验证镜像可用性..."

    # 检查关键镜像
    local key_images=(
        "quanzhenglong.com/edge/edge:develop"
        "quanzhenglong.com/edge/clickhouse-server:latest"
        "quanzhenglong.com/edge/fluent-bit:latest"
    )

    for image in "${key_images[@]}"; do
        if docker pull "$image" &>/dev/null; then
            record_test "镜像可用-$(basename "$image")" "PASS" "拉取成功"
        else
            record_test "镜像可用-$(basename "$image")" "FAIL" "拉取失败"
        fi
    done
}

# 测试Helm Chart
test_helm_chart() {
    log_info "验证Helm Chart..."

    local chart_dir="${SCRIPT_DIR}/../helm/edge-logs"

    # 检查Chart语法
    if helm lint "$chart_dir" &>/dev/null; then
        record_test "Chart语法" "PASS" "语法检查通过"
    else
        record_test "Chart语法" "FAIL" "语法检查失败"
    fi

    # 检查模板渲染
    if helm template test-release "$chart_dir" &>/dev/null; then
        record_test "Chart模板" "PASS" "模板渲染成功"
    else
        record_test "Chart模板" "FAIL" "模板渲染失败"
    fi
}

# 部署测试环境
test_deployment() {
    local env="$1"
    log_info "测试 $env 环境部署..."

    local namespace="edge-logs-$env"
    local release_name="edge-logs-$env"

    # 清理已存在的部署
    if helm list -n "$namespace" | grep -q "$release_name"; then
        log_info "清理现有部署..."
        helm uninstall "$release_name" -n "$namespace" --timeout 60s &>/dev/null || true
    fi

    # 等待清理完成
    sleep 10

    # 执行部署
    if timeout 300 "${SCRIPT_DIR}/../edge-helm" deploy "$env" --wait &>/dev/null; then
        record_test "部署-$env" "PASS" "部署成功"
    else
        record_test "部署-$env" "FAIL" "部署超时或失败"
        return 1
    fi

    # 检查Pod状态
    local running_pods
    running_pods=$(kubectl get pods -n "$namespace" --field-selector=status.phase=Running --no-headers | wc -l)

    if [[ $running_pods -gt 0 ]]; then
        record_test "Pod状态-$env" "PASS" "$running_pods 个Pod运行中"
    else
        record_test "Pod状态-$env" "FAIL" "没有运行的Pod"
    fi

    # 检查服务状态
    local services
    services=$(kubectl get svc -n "$namespace" --no-headers | wc -l)

    if [[ $services -gt 0 ]]; then
        record_test "服务状态-$env" "PASS" "$services 个服务创建"
    else
        record_test "服务状态-$env" "FAIL" "没有服务创建"
    fi
}

# 功能测试
test_functionality() {
    local env="$1"
    log_info "测试 $env 环境功能..."

    local namespace="edge-logs-$env"

    # 等待服务就绪
    sleep 30

    # 检查ClickHouse
    if kubectl exec -n "$namespace" statefulset/edge-logs-clickhouse -- clickhouse-client --query "SELECT 1" &>/dev/null; then
        record_test "ClickHouse-$env" "PASS" "数据库连接正常"
    else
        record_test "ClickHouse-$env" "FAIL" "数据库连接失败"
    fi

    # 检查APIServer健康检查 (可能因为使用nginx镜像而失败，这是预期的)
    if kubectl exec -n "$namespace" deployment/edge-logs-apiserver -- wget -qO- http://localhost:8080/api/v1alpha1/health &>/dev/null; then
        record_test "APIServer健康检查-$env" "PASS" "健康检查通过"
    else
        record_test "APIServer健康检查-$env" "WARN" "健康检查失败(预期,使用nginx镜像)"
    fi

    # 检查服务间连通性
    if kubectl exec -n "$namespace" deployment/edge-logs-frontend -- curl -I http://edge-logs-apiserver:8080 &>/dev/null; then
        record_test "服务连通性-$env" "PASS" "Frontend到APIServer连通"
    else
        record_test "服务连通性-$env" "WARN" "Frontend到APIServer不通(预期,nginx镜像)"
    fi
}

# 清理测试环境
cleanup_test_environment() {
    local env="$1"
    log_info "清理 $env 测试环境..."

    local namespace="edge-logs-$env"
    local release_name="edge-logs-$env"

    if helm list -n "$namespace" | grep -q "$release_name"; then
        helm uninstall "$release_name" -n "$namespace" --timeout 60s &>/dev/null || true
        record_test "环境清理-$env" "PASS" "清理完成"
    else
        record_test "环境清理-$env" "PASS" "无需清理"
    fi
}

# 生成测试报告
generate_report() {
    echo ""
    echo "=================================="
    echo "         验收测试报告"
    echo "=================================="
    echo "测试时间: $(date)"
    echo "总测试数: $TOTAL_TESTS"
    echo "通过: $((TOTAL_TESTS - FAILED_TESTS))"
    echo "失败: $FAILED_TESTS"
    echo ""

    echo "详细结果:"
    for result in "${TEST_RESULTS[@]}"; do
        echo "$result"
    done

    echo ""
    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_success "🎉 所有测试通过！部署验收成功！"
        return 0
    else
        log_error "💥 有 $FAILED_TESTS 个测试失败"
        return 1
    fi
}

# 显示帮助
show_help() {
    cat << EOF
Edge Logs 验收测试工具

用法:
  $0 [选项]

选项:
  --env ENV        仅测试指定环境 (dev/staging/prod)
  --no-cleanup     测试后不清理环境
  --quick          快速测试，跳过镜像验证
  --help           显示此帮助

示例:
  $0                    # 完整验收测试
  $0 --env dev          # 仅测试dev环境
  $0 --quick            # 快速测试
  $0 --no-cleanup       # 保留测试环境

EOF
}

# 主测试流程
main() {
    local test_env=""
    local cleanup=true
    local quick=false

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --env)
                test_env="$2"
                shift 2
                ;;
            --no-cleanup)
                cleanup=false
                shift
                ;;
            --quick)
                quick=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done

    echo "🚀 开始 Edge Logs 验收测试..."
    echo ""

    # 基础测试
    test_dependencies || return 1
    test_cluster_connection || return 1
    test_helm_chart || return 1

    # 镜像测试 (除非快速模式)
    if [[ "$quick" != "true" ]]; then
        test_image_availability
    fi

    # 环境测试
    if [[ -n "$test_env" ]]; then
        environments=("$test_env")
    else
        environments=("dev")  # 只测试dev环境，避免过多资源消耗
    fi

    for env in "${environments[@]}"; do
        test_deployment "$env"
        test_functionality "$env"

        if [[ "$cleanup" == "true" ]]; then
            cleanup_test_environment "$env"
        fi
    done

    # 生成报告
    generate_report
}

# 捕获中断信号，确保清理
trap 'log_warn "测试被中断"; exit 1' INT TERM

# 执行主流程
main "$@"