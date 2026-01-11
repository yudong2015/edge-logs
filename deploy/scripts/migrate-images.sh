#!/bin/bash
# Edge Logs 镜像迁移脚本
# 将所有依赖镜像复制到国内仓库

set -e

# 配置
SOURCE_REGISTRY=""  # Docker Hub 默认
TARGET_REGISTRY="quanzhenglong.com/edge"

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
    log_info "检查必要工具..."

    if ! command -v skopeo &> /dev/null; then
        log_error "skopeo 未安装，请安装 skopeo"
        echo "Ubuntu/Debian: sudo apt install skopeo"
        echo "RHEL/CentOS: sudo yum install skopeo"
        echo "macOS: brew install skopeo"
        exit 1
    fi

    log_success "工具检查完成"
}

# 镜像映射定义
declare -a SOURCE_IMAGES=(
    "clickhouse/clickhouse-server:latest"
    "clickhouse/clickhouse-server:24.3"
    "fluent/fluent-bit:latest"
    "fluent/fluent-bit:3.1.9"
    "grafana/promtail:2.8.2"
    "grafana/promtail:latest"
    "prom/prometheus:latest"
    "grafana/grafana:latest"
    "nginx:latest"
    "nginx:1.25"
    "busybox:latest"
    "alpine:latest"
)

declare -a TARGET_IMAGES=(
    "clickhouse-server:latest"
    "clickhouse-server:24.3"
    "fluent-bit:latest"
    "fluent-bit:3.1.9"
    "promtail:2.8.2"
    "promtail:latest"
    "prometheus:latest"
    "grafana:latest"
    "nginx:latest"
    "nginx:1.25"
    "busybox:latest"
    "alpine:latest"
)

# 复制单个镜像
copy_image() {
    local source="$1"
    local target="$2"

    log_info "复制镜像: $source → $TARGET_REGISTRY/$target"

    # 检查源镜像是否存在
    if ! skopeo inspect "docker://$source" &>/dev/null; then
        log_warn "源镜像不存在，跳过: $source"
        return 0
    fi

    # 复制镜像
    if skopeo copy --all \
        "docker://$source" \
        "docker://$TARGET_REGISTRY/$target" \
        --retry-times 3; then
        log_success "复制成功: $target"
        return 0
    else
        log_error "复制失败: $source"
        return 1
    fi
}

# 验证镜像
verify_image() {
    local image="$1"

    log_info "验证镜像: $TARGET_REGISTRY/$image"

    if skopeo inspect "docker://$TARGET_REGISTRY/$image" &>/dev/null; then
        log_success "验证成功: $image"
        return 0
    else
        log_error "验证失败: $image"
        return 1
    fi
}

# 生成镜像清单
generate_manifest() {
    local manifest_file="image-manifest.yaml"

    log_info "生成镜像清单: $manifest_file"

    cat > "$manifest_file" << EOF
# Edge Logs 镜像清单
# 生成时间: $(date)
# 目标仓库: $TARGET_REGISTRY

images:
EOF

    for i in "${!SOURCE_IMAGES[@]}"; do
        local source="${SOURCE_IMAGES[$i]}"
        local target="${TARGET_IMAGES[$i]}"
        echo "  - source: $source" >> "$manifest_file"
        echo "    target: $TARGET_REGISTRY/$target" >> "$manifest_file"
        echo "    status: $(skopeo inspect "docker://$TARGET_REGISTRY/$target" &>/dev/null && echo "available" || echo "missing")" >> "$manifest_file"
        echo "" >> "$manifest_file"
    done

    log_success "镜像清单已生成: $manifest_file"
}

# 主执行函数
main() {
    log_info "开始镜像迁移过程..."

    check_dependencies

    # 统计信息
    local total=${#SOURCE_IMAGES[@]}
    local success_count=0
    local failed_count=0

    log_info "总共需要迁移 $total 个镜像"

    # 逐个复制镜像
    for i in "${!SOURCE_IMAGES[@]}"; do
        local source="${SOURCE_IMAGES[$i]}"
        local target="${TARGET_IMAGES[$i]}"

        if copy_image "$source" "$target"; then
            ((success_count++))
        else
            ((failed_count++))
        fi

        echo "进度: $((success_count + failed_count))/$total"
        echo ""
    done

    # 验证阶段
    log_info "开始验证已复制的镜像..."
    local verify_success=0

    for target in "${TARGET_IMAGES[@]}"; do
        if verify_image "$target"; then
            ((verify_success++))
        fi
    done

    # 生成清单
    generate_manifest

    # 结果统计
    echo ""
    echo "======================"
    echo "   迁移结果统计"
    echo "======================"
    echo "总镜像数量: $total"
    echo "成功复制: $success_count"
    echo "复制失败: $failed_count"
    echo "验证通过: $verify_success"
    echo ""

    if [[ $failed_count -eq 0 && $verify_success -eq $total ]]; then
        log_success "✅ 所有镜像迁移成功！"
        return 0
    else
        log_error "❌ 部分镜像迁移失败，请检查日志"
        return 1
    fi
}

# 显示帮助
show_help() {
    cat << EOF
Edge Logs 镜像迁移工具

用法:
  $0 [选项]

选项:
  --dry-run          仅显示将要执行的操作，不实际复制
  --verify-only      仅验证目标镜像是否存在
  --list             列出所有镜像映射关系
  --help             显示此帮助

示例:
  $0                 # 执行完整镜像迁移
  $0 --dry-run       # 预览操作
  $0 --verify-only   # 验证镜像

环境变量:
  TARGET_REGISTRY    目标镜像仓库 (默认: quanzhenglong.com/edge)

EOF
}

# 命令行参数处理
case "${1:-}" in
    --help|-h)
        show_help
        exit 0
        ;;
    --dry-run)
        log_info "DRY-RUN 模式，仅预览操作"
        for i in "${!SOURCE_IMAGES[@]}"; do
            source="${SOURCE_IMAGES[$i]}"
            target="${TARGET_IMAGES[$i]}"
            echo "将复制: $source → $TARGET_REGISTRY/$target"
        done
        exit 0
        ;;
    --verify-only)
        log_info "仅验证模式"
        for target in "${TARGET_IMAGES[@]}"; do
            verify_image "$target"
        done
        exit 0
        ;;
    --list)
        log_info "镜像映射关系:"
        for i in "${!SOURCE_IMAGES[@]}"; do
            source="${SOURCE_IMAGES[$i]}"
            target="${TARGET_IMAGES[$i]}"
            echo "$source → $TARGET_REGISTRY/$target"
        done
        exit 0
        ;;
    "")
        main
        ;;
    *)
        log_error "未知选项: $1"
        show_help
        exit 1
        ;;
esac