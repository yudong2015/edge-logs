# Edge Logs 国产化部署指南

## 🎯 目标

实现 Edge Logs 在国内服务器的快速、稳定部署，解决网络访问和镜像拉取问题。

## 📋 准备工作

### 1. 环境要求

- **Kubernetes集群**: v1.20+
- **Helm**: v3.0+
- **Docker/Podman**: 用于镜像操作
- **skopeo**: 用于镜像迁移

### 2. 网络配置

确保集群能够访问国内镜像仓库：
```bash
# 测试仓库连通性
docker pull quanzhenglong.com/edge/nginx:latest
```

## 🚀 部署流程

### 步骤1: 镜像迁移 (首次部署)

```bash
cd deploy/scripts

# 查看需要迁移的镜像
./migrate-images.sh --list

# 执行镜像迁移 (需要仓库推送权限)
./migrate-images.sh

# 验证镜像迁移结果
./migrate-images.sh --verify-only
```

### 步骤2: 验证镜像可用性

```bash
# 验证开发环境镜像
./verify-images.sh dev

# 验证所有环境镜像
./verify-images.sh all
```

### 步骤3: 部署应用

```bash
cd ../

# 部署到开发环境
./edge-helm deploy dev

# 部署到预发环境
./edge-helm deploy staging

# 部署到生产环境 (启用采集器)
./edge-helm deploy prod
```

### 步骤4: 验证部署结果

```bash
# 检查部署状态
./edge-helm status dev

# 查看Pod状态
kubectl get pods -n edge-logs-dev -o wide

# 检查镜像使用情况
kubectl get pods -n edge-logs-dev -o jsonpath='{.items[*].spec.containers[*].image}' | tr ' ' '\n' | sort -u

# 运行健康检查
./edge-helm health dev
```

## 📊 国产化方案详情

### 镜像仓库映射

| 原始镜像 | 国内镜像 | 用途 |
|---------|---------|------|
| `clickhouse/clickhouse-server:latest` | `quanzhenglong.com/edge/clickhouse-server:latest` | 数据库 |
| `fluent/fluent-bit:latest` | `quanzhenglong.com/edge/fluent-bit:latest` | 日志采集 |
| `nginx:latest` | `quanzhenglong.com/edge/nginx:latest` | Web服务 |
| `grafana/promtail:2.8.2` | `quanzhenglong.com/edge/promtail:2.8.2` | 日志转发 |

### 自研镜像构建

应用镜像通过GitHub Actions自动构建：

```yaml
# GitHub Actions触发条件
推送到main分支     → quanzhenglong.com/edge:latest
推送到develop分支  → quanzhenglong.com/edge:develop
创建v1.0.0 tag   → quanzhenglong.com/edge:v1.0.0
```

### Helm Chart配置

全局镜像仓库配置：
```yaml
# values.yaml
global:
  imageRegistry: "quanzhenglong.com/edge"

# 各组件使用空registry，自动使用global配置
apiserver:
  image:
    registry: ""  # 使用global.imageRegistry
    repository: edge
    tag: "develop"
```

## 🔧 故障排查

### 常见问题

#### 1. 镜像拉取失败

**症状**:
```
Failed to pull image "quanzhenglong.com/edge/xxx": Error response from daemon: pull access denied
```

**解决方案**:
```bash
# 检查镜像是否存在
./verify-images.sh dev --list

# 确认网络连通性
docker pull quanzhenglong.com/edge/nginx:latest

# 重新迁移缺失的镜像
./migrate-images.sh
```

#### 2. Pod启动失败

**症状**:
```
Pod edge-logs-apiserver-xxx is in CrashLoopBackOff state
```

**解决方案**:
```bash
# 查看Pod详情
kubectl describe pod <pod-name> -n edge-logs-dev

# 查看日志
./edge-helm logs dev --component apiserver

# 检查配置
helm get values edge-logs-dev -n edge-logs-dev
```

#### 3. 服务连通性问题

**症状**:
```
Connection refused to edge-logs-apiserver
```

**解决方案**:
```bash
# 检查服务状态
kubectl get svc -n edge-logs-dev

# 测试服务连通性
kubectl exec -n edge-logs-dev deployment/edge-logs-frontend -- curl -I http://edge-logs-apiserver:8080/api/v1alpha1/health

# 检查网络策略
kubectl get networkpolicies -n edge-logs-dev
```

### 性能优化

#### 1. 镜像拉取优化

```bash
# 使用本地镜像缓存
docker system df

# 清理无用镜像
docker image prune -f

# 预拉取关键镜像
docker pull quanzhenglong.com/edge/edge:latest
docker pull quanzhenglong.com/edge/clickhouse-server:latest
```

#### 2. 存储优化

```yaml
# 为不同环境配置合适的存储大小
dev环境:     5Gi   (开发测试)
staging环境: 20Gi  (预发验证)
prod环境:    100Gi (生产运行)
```

## 📋 验收标准

### 必须满足的条件

- [ ] 所有镜像成功迁移到 `quanzhenglong.com/edge`
- [ ] 三个环境 (dev/staging/prod) 部署无错误
- [ ] 所有Pod状态为 Running
- [ ] 健康检查全部通过
- [ ] 服务间连通性正常
- [ ] 日志采集器正常工作 (prod环境)

### 验收命令

```bash
# 完整验收流程
cd deploy/scripts

# 1. 验证镜像
./verify-images.sh all

# 2. 部署所有环境
cd ..
./edge-helm deploy dev --wait
./edge-helm deploy staging --wait
./edge-helm deploy prod --wait

# 3. 健康检查
./edge-helm health dev
./edge-helm health staging
./edge-helm health prod

# 4. 功能验证
kubectl exec -n edge-logs deployment/edge-logs-frontend -- curl -sf http://edge-logs-apiserver:8080/api/v1alpha1/health
```

## 📈 监控和维护

### 日常操作

```bash
# 查看资源使用
kubectl top pods -n edge-logs-dev

# 查看日志
./edge-helm logs dev --component apiserver

# 更新部署
./edge-helm upgrade dev --tag v1.0.1

# 回滚版本
./edge-helm rollback dev
```

### 镜像更新流程

1. **代码提交** → GitHub Actions自动构建新镜像
2. **验证镜像** → `./verify-images.sh dev`
3. **升级部署** → `./edge-helm upgrade dev --tag new-version`
4. **验证功能** → `./edge-helm health dev`

---

**部署完成标志**: 所有验收标准✅，系统稳定运行，无错误日志

**维护建议**: 定期检查镜像更新，监控集群资源使用，备份关键数据