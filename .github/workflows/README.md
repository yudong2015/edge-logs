# GitHub Actions 配置指南

## 环境变量配置

在GitHub仓库的 **Settings → Secrets and variables → Actions** 中配置以下变量：

### Repository Variables (公开变量)

```bash
DOCKER_REGISTRY=quanzhenglong.com    # 国内镜像仓库
DOCKER_REPO=edge                     # 镜像仓库名称
DOCKER_USERNAME=your_username        # 仓库用户名
```

### Repository Secrets (私密变量)

```bash
DOCKER_PASSWORD=your_password        # 仓库密码
```

## 镜像推送规则

### 自动触发条件
- Push到 `main`, `develop`, `feature/**`, `hotfix/**` 分支
- 创建 `v*` 格式的tag
- 创建针对 `main`, `develop` 的PR

### 镜像标签规则
```bash
# 分支推送
main分支     → quanzhenglong.com/edge:latest
develop分支  → quanzhenglong.com/edge:develop
feature/xxx  → quanzhenglong.com/edge:feature-xxx

# 版本发布
v1.0.0 tag  → quanzhenglong.com/edge:v1.0.0
v1.0.1 tag  → quanzhenglong.com/edge:v1.0.1

# commit标签
任意分支     → quanzhenglong.com/edge:{branch}-{sha7}
```

### 构建产物
- **APIServer**: `quanzhenglong.com/edge:{tag}`
- **Frontend**: `quanzhenglong.com/edge-frontend:{tag}`
- **平台支持**: linux/amd64, linux/arm64

## 部署使用

```bash
# 部署develop版本到开发环境
./edge-helm deploy dev

# 部署指定版本到生产环境
./edge-helm deploy prod --tag v1.0.0

# 查看可用镜像
docker search quanzhenglong.com/edge
```

## 故障排查

### 构建失败
1. 检查Docker Registry连通性
2. 确认认证信息正确
3. 查看GitHub Actions日志

### 镜像拉取失败
1. 确认镜像标签存在
2. 检查网络连通性到 quanzhenglong.com
3. 验证认证配置

## 安全注意事项

1. **密码保护**: `DOCKER_PASSWORD` 必须设置为Secret
2. **网络安全**: 仅信任的仓库可以推送
3. **镜像扫描**: CI中集成了Trivy安全扫描
4. **权限控制**: 只有维护者可以修改镜像

---
更新时间: 2026-01-11