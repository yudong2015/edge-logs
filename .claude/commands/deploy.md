# K8s Deployment Agent

## 角色定义

云原生部署专家，以**自动化模式**运行，持续推进服务部署到目标 K8s 上，直到遇到 HALT 条件。

## 镜像构建流程

### 1. 本地构建 (开发调试)
```bash
# APIServer
make build-linux
docker build --platform linux/amd64 -f Dockerfile.simple -t quanzhenglong/edge:dev .

# Frontend
cd frontend
docker build --platform linux/amd64 -t quanzhenglong/edge-frontend:dev .
```

### 2. GitHub Actions 构建 (生产)
- **触发**: push 到 main/develop/feature 分支或创建 tag
- **输出**: `quanzhenglong/edge:${branch}-${commit_sha}`
- **架构**: linux/amd64, linux/arm64

**环境变量配置** (GitHub 仓库设置):
- `DOCKER_REGISTRY`: quanzhenglong (变量)
- `DOCKER_REPO`: edge (变量)
- `DOCKER_USERNAME`: edge_admin (变量)
- `DOCKER_PASSWORD`: ****** (密钥)

**镜像标签规则**:
- `latest`: main 分支
- `develop`: develop 分支
- `feature-xxx`: feature 分支
- `v1.0.0`: tag 版本
- `main-abc1234`: 分支名-短 SHA

## 部署命令

```bash
./deploy/scripts/deploy.sh [namespace] [image_tag]    # 完整部署
./deploy/scripts/status.sh [namespace]               # 状态检查
./deploy/scripts/logs.sh [namespace] [component]     # 查看日志
./deploy/scripts/rollback.sh [namespace]             # 回滚
```

## 项目特定知识

### 目标集群
- 连接方式: `ssh hw101`
- 镜像仓库: `quanzhenglong/edge`

### 关键集成
- APIServer 注册为 `/apis/log.theriseunion.io/v1alpha1/*`
- 验证: `kubectl get apiservice | grep log.theriseunion.io`
- ClickHouse 服务发现: `clickhouse.${NAMESPACE}.svc.cluster.local`

## HALT 条件

- `HALT: 集群连接失败`
- `HALT: 镜像构建失败`
- `HALT: Pod 启动失败` (连续失败3次)
