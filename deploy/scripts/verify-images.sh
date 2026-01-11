#!/bin/bash
# 验证所有部署需要的镜像是否可用

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../helm/edge-logs"

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

# 检查依赖
check_dependencies() {
    if ! command -v helm &> /dev/null; then
        log_error "helm 未安装"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_error "docker 未安装"
        exit 1
    fi
}

# 从Helm模板提取镜像
extract_images() {
    local env="${1:-dev}"
    local values_file="${SCRIPT_DIR}/../helm/values-${env}.yaml"

    log_info "提取 $env 环境的镜像列表..."

    if [[ ! -f "$values_file" ]]; then
        log_error "环境配置文件不存在: $values_file"
        exit 1
    fi

    # 生成模板并提取镜像
    helm template test-release "$CHART_DIR" \
        -f "$values_file" \
        --set logCollector.enabled=true \
        2>/dev/null | \
        grep -E "^\s*image:" | \
        sed 's/.*image:\s*//' | \
        sort -u
}

# 验证单个镜像
verify_image() {
    local image="$1"

    log_info "验证镜像: $image"

    # 尝试拉取镜像
    if docker pull "$image" &>/dev/null; then
        log_success "✅ $image"
        return 0
    else
        log_error "❌ $image"
        return 1
    fi
}

# 主验证函数
verify_environment() {
    local env="$1"
    local success_count=0
    local failed_count=0

    log_info "开始验证 $env 环境镜像..."
    echo ""

    # 提取镜像列表
    local images
    images=$(extract_images "$env")

    if [[ -z "$images" ]]; then
        log_error "未找到任何镜像"
        return 1
    fi

    # 逐个验证镜像
    while IFS= read -r image; do
        if verify_image "$image"; then
            ((success_count++))
        else
            ((failed_count++))
        fi
    done <<< "$images"

    # 结果统计
    echo ""
    echo "=================="
    echo "  验证结果 ($env)"
    echo "=================="
    echo "成功: $success_count"
    echo "失败: $failed_count"
    echo ""

    if [[ $failed_count -eq 0 ]]; then
        log_success "✅ $env 环境所有镜像验证通过"
        return 0
    else
        log_error "❌ $env 环境有 $failed_count 个镜像验证失败"
        return 1
    fi
}

# 显示帮助
show_help() {
    cat << EOF
Edge Logs 镜像验证工具

用法:
  $0 [环境] [选项]

环境:
  dev         验证开发环境镜像 (默认)
  staging     验证预发环境镜像
  prod        验证生产环境镜像
  all         验证所有环境镜像

选项:
  --list      仅列出镜像，不验证
  --help      显示此帮助

示例:
  $0                    # 验证dev环境
  $0 prod               # 验证生产环境
  $0 all                # 验证所有环境
  $0 dev --list         # 列出dev环境镜像

EOF
}

# 主函数
main() {
    local env="${1:-dev}"
    local list_only=false

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --list)
                list_only=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            dev|staging|prod|all)
                env="$1"
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done

    check_dependencies

    if [[ "$list_only" == "true" ]]; then
        if [[ "$env" == "all" ]]; then
            for e in dev staging prod; do
                echo "=== $e 环境镜像 ==="
                extract_images "$e"
                echo ""
            done
        else
            echo "=== $env 环境镜像 ==="
            extract_images "$env"
        fi
        exit 0
    fi

    # 执行验证
    if [[ "$env" == "all" ]]; then
        local overall_success=true
        for e in dev staging prod; do
            if ! verify_environment "$e"; then
                overall_success=false
            fi
        done

        if [[ "$overall_success" == "true" ]]; then
            log_success "🎉 所有环境镜像验证通过！"
            exit 0
        else
            log_error "💥 部分环境镜像验证失败"
            exit 1
        fi
    else
        verify_environment "$env"
    fi
}

main "$@"