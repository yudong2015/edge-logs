# Edge Logs 部署文档

## 快速开始

### 1. 设置 GitHub Actions 环境变量

在 GitHub 仓库的 Settings > Secrets and variables > Actions 中设置：

**变量 (Variables)**:
- `DOCKER_REGISTRY`: `quanzhenglong`
- `DOCKER_REPO`: `edge`
- `DOCKER_USERNAME`: `edge_admin`

**密钥 (Secrets)**:
- `DOCKER_PASSWORD`: 密码

### 2. 部署到开发环境

```bash
cd deploy
make deploy ENV=dev
```

### 3. 检查部署状态

```bash
make status ENV=dev
```

## 详细使用

### 部署命令

```bash
# 使用 Makefile (推荐)
make deploy ENV=dev                    # 部署开发环境
make deploy ENV=staging IMAGE_TAG=v1.0.0  # 部署预发，指定版本
make status ENV=dev                    # 查看状态
make logs ENV=dev COMPONENT=apiserver  # 查看日志
make rollback ENV=dev                  # 回滚

# 直接使用脚本
./scripts/deploy.sh dev main-abc1234   # 部署开发环境
./scripts/status.sh dev                # 查看状态
./scripts/logs.sh dev apiserver        # 查看 APIServer 日志
./scripts/rollback.sh dev all          # 回滚所有组件
```

### 环境说明

| 环境 | 命名空间 | 镜像标签 | 说明 |
|------|----------|----------|------|
| dev | edge-logs-dev | develop | 开发环境 |
| staging | edge-logs-staging | main | 预发环境 |
| prod | edge-logs | latest | 生产环境 |

### 组件说明

| 组件 | 端口 | 健康检查 |
|------|------|----------|
| ClickHouse | 8123, 9000 | `/ping` |
| APIServer | 8080 | `/api/v1alpha1/health` |
| Frontend | 80 | `/healthz` |

## CI/CD 流程

### 镜像构建

1. **触发条件**:
   - Push 到 main/develop/feature 分支
   - 创建 tag (如 v1.0.0)
   - 创建 PR

2. **镜像标签规则**:
   - `latest`: main 分支
   - `develop`: develop 分支
   - `feature-xxx`: feature 分支
   - `v1.0.0`: tag 版本
   - `main-abc1234`: 分支名-短 SHA

3. **多架构支持**: linux/amd64, linux/arm64

### 部署流程

1. **代码提交** → GitHub Actions 自动构建镜像
2. **镜像推送** → 国内镜像仓库 (解决拉取速度问题)
3. **手动部署** → 使用 `make deploy` 或脚本部署
4. **健康检查** → 自动验证部署状态

## 故障排查

### 常见问题

1. **镜像拉取失败**
   ```bash
   # 检查镜像是否存在
   docker pull quanzhenglong/edge:latest

   # 查看 Pod 事件
   kubectl describe pod -n edge-logs-dev
   ```

2. **Pod 启动失败**
   ```bash
   # 查看日志
   make logs ENV=dev COMPONENT=apiserver

   # 查看 Pod 状态
   kubectl get pods -n edge-logs-dev -o wide
   ```

3. **服务连通性问题**
   ```bash
   # 测试服务连接
   kubectl exec -n edge-logs-dev deployment/edge-logs-frontend -- \
     curl -v http://edge-logs-apiserver.edge-logs-dev.svc.cluster.local:8080/api/v1alpha1/health
   ```

### 手动操作

```bash
# 强制重新拉取镜像
kubectl rollout restart deployment/edge-logs-apiserver -n edge-logs-dev

# 查看资源使用情况
kubectl top pods -n edge-logs-dev

# 进入容器调试
kubectl exec -it deployment/edge-logs-apiserver -n edge-logs-dev -- sh
```

## 开发建议

### 快速迭代

1. **修改代码** → 提交到 develop 分支
2. **等待 CI** → GitHub Actions 自动构建 `develop` 标签镜像
3. **快速部署** → `make deploy ENV=dev`
4. **实时日志** → `make logs ENV=dev COMPONENT=apiserver`

### 版本发布

1. **创建 tag** → `git tag v1.0.0 && git push origin v1.0.0`
2. **CI 构建** → 自动构建 `v1.0.0` 镜像
3. **部署预发** → `make deploy ENV=staging IMAGE_TAG=v1.0.0`
4. **验证功能** → 测试验证
5. **发布生产** → `make deploy ENV=prod IMAGE_TAG=v1.0.0`

这样设置后，开发效率将大大提升，镜像构建时间从本地的几分钟缩短到 GitHub Actions 的并行构建，部署时间也从手动操作缩短到一条命令。