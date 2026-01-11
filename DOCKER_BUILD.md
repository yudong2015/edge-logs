# Edge Logs 镜像构建指南

## 📦 镜像构建流程

### 自动化构建 (推荐)

**GitHub Actions 自动构建**
- 推送到分支时自动触发构建
- 支持多架构: `linux/amd64`, `linux/arm64`
- 自动推送到镜像仓库: `quanzhenglong/edge`

**镜像标签规则**:
```bash
# 分支构建
main分支     → quanzhenglong/edge:latest
develop分支  → quanzhenglong/edge:develop
feature/*    → quanzhenglong/edge:feature-xxx

# 版本发布
v1.0.0 tag  → quanzhenglong/edge:v1.0.0
v1.0.1 tag  → quanzhenglong/edge:v1.0.1

# commit构建
任意分支     → quanzhenglong/edge:{branch}-{sha7}
```

### 本地构建 (开发调试)

#### 1. APIServer 镜像

```bash
# 基础构建
docker build -f deploy/apiserver/Dockerfile -t edge-logs-apiserver:local .

# 多架构构建
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f deploy/apiserver/Dockerfile \
  -t quanzhenglong/edge:local \
  --push .

# 带版本信息构建
docker build \
  -f deploy/apiserver/Dockerfile \
  --build-arg VERSION=v0.1.0 \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  -t edge-logs-apiserver:v0.1.0 .
```

#### 2. Frontend 镜像 (如果存在)

```bash
# 构建Frontend镜像
cd frontend
docker build -t edge-logs-frontend:local .

# 推送到仓库
docker tag edge-logs-frontend:local quanzhenglong/edge-frontend:local
docker push quanzhenglong/edge-frontend:local
```

#### 3. 快速构建脚本

```bash
#!/bin/bash
# build-images.sh

VERSION="${1:-local}"
REGISTRY="${REGISTRY:-quanzhenglong}"

echo "构建版本: $VERSION"

# 构建APIServer
docker build \
  -f deploy/apiserver/Dockerfile \
  -t "${REGISTRY}/edge:${VERSION}" \
  --build-arg VERSION="$VERSION" \
  --build-arg GIT_COMMIT="$(git rev-parse HEAD)" \
  --build-arg BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
  .

# 构建Frontend (如果存在)
if [ -d "frontend" ]; then
  docker build \
    -f frontend/Dockerfile \
    -t "${REGISTRY}/edge-frontend:${VERSION}" \
    frontend/
fi

echo "构建完成!"
echo "APIServer: ${REGISTRY}/edge:${VERSION}"
echo "Frontend: ${REGISTRY}/edge-frontend:${VERSION}"
```

## 🚀 部署使用镜像

### 开发环境

```bash
# 使用develop标签部署
./edge-helm deploy dev

# 使用自定义标签
./edge-helm deploy dev --tag feature-auth

# 使用本地构建的镜像
./edge-helm deploy dev --tag local
```

### 预发环境

```bash
# 使用main分支镜像
./edge-helm deploy staging

# 使用特定版本
./edge-helm deploy staging --tag v1.0.0
```

### 生产环境

```bash
# 使用latest标签 (main分支)
./edge-helm deploy prod

# 使用指定版本 (推荐)
./edge-helm deploy prod --tag v1.0.1
```

## 📝 Docker构建优化

### 多阶段构建优势

```dockerfile
# 阶段1: 构建环境
FROM golang:1.24-alpine AS builder
# ... 编译Go应用

# 阶段2: 运行环境
FROM scratch
# ... 只包含必要的运行文件
```

**优势**:
- 镜像体积小 (最终镜像 < 20MB)
- 安全性高 (无操作系统漏洞)
- 启动速度快

### 构建缓存优化

```dockerfile
# 先复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 再复制源码 (利用Docker层缓存)
COPY . .
RUN go build ...
```

### BuildKit 缓存

```bash
# 启用BuildKit缓存挂载
docker build \
  --build-arg BUILDKIT_INLINE_CACHE=1 \
  -t edge-logs:latest .

# GitHub Actions中使用缓存
- name: Build and push
  uses: docker/build-push-action@v5
  with:
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

## 🔍 镜像调试

### 查看镜像信息

```bash
# 镜像详情
docker inspect quanzhenglong/edge:develop

# 镜像历史
docker history quanzhenglong/edge:develop

# 镜像体积
docker images quanzhenglong/edge
```

### 运行调试

```bash
# 本地运行APIServer
docker run -it --rm \
  -p 8080:8080 \
  -e CLICKHOUSE_HOST=host.docker.internal \
  quanzhenglong/edge:develop

# 进入容器调试
docker run -it --rm \
  --entrypoint /bin/sh \
  quanzhenglong/edge:develop

# 查看应用日志
docker logs <container_id>
```

### 安全扫描

```bash
# 使用Trivy扫描漏洞
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasecurity/trivy image quanzhenglong/edge:develop

# 查看镜像签名
docker trust inspect quanzhenglong/edge:develop
```

## 📊 CI/CD 集成

### GitHub Actions 构建

```yaml
# .github/workflows/build-and-push.yml
name: Build and Push Images

on:
  push:
    branches: [main, develop, 'feature/**']
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ vars.DOCKER_REGISTRY }}
          username: ${{ vars.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./deploy/apiserver/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            quanzhenglong/edge:${{ github.ref_name }}
            quanzhenglong/edge:${{ github.ref_name }}-${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### 部署触发

```bash
# 代码提交后自动构建
git push origin develop
# → 触发构建 quanzhenglong/edge:develop

# 本地部署使用新镜像
./edge-helm deploy dev
# → 自动拉取 quanzhenglong/edge:develop

# 版本发布
git tag v1.0.0 && git push origin v1.0.0
# → 构建 quanzhenglong/edge:v1.0.0

# 部署生产版本
./edge-helm deploy prod --tag v1.0.0
```

## 🔐 镜像安全最佳实践

### 1. 基础镜像选择

```dockerfile
# ✅ 推荐: 使用最小化基础镜像
FROM scratch
FROM alpine:3.18
FROM distroless/static

# ❌ 避免: 使用大型基础镜像
FROM ubuntu:latest
FROM centos:latest
```

### 2. 用户权限

```dockerfile
# 创建非root用户
USER 1000:1000

# 或在Kubernetes中设置
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
```

### 3. 镜像签名

```bash
# 启用Docker Content Trust
export DOCKER_CONTENT_TRUST=1

# 签名并推送
docker trust sign quanzhenglong/edge:v1.0.0
```

### 4. 漏洞扫描

```bash
# CI/CD中集成安全扫描
docker run --rm \
  aquasecurity/trivy image \
  --exit-code 1 \
  --severity HIGH,CRITICAL \
  quanzhenglong/edge:$TAG
```

---

**镜像仓库信息**:
- Registry: `quanzhenglong` (Docker Hub)
- Repository: `edge` (APIServer), `edge-frontend` (Frontend)
- 访问方式: 公开拉取，推送需要认证

**更新记录**:
- 2026-01-11: 初始版本，包含完整的构建和部署流程