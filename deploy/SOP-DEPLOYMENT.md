# Edge Logs K8s 部署标准操作程序 (SOP)

## 📋 概述

本SOP文档详细说明了Edge Logs系统在Kubernetes集群中的标准部署流程，基于实际部署经验整理。

## ✅ 先决条件检查

### 1. 环境要求
- **K8s集群**: v1.30+
- **kubectl**: 已配置并可访问目标集群
- **工具依赖**: `envsubst` (gettext包)
- **权限要求**: 集群管理员权限或相应namespace权限

### 2. 镜像要求
- **APIServer镜像**: `{DOCKER_REGISTRY}/{DOCKER_REPO}:{IMAGE_TAG}`
- **Frontend镜像**: `{DOCKER_REGISTRY}/{DOCKER_REPO}-frontend:{IMAGE_TAG}`
- **ClickHouse镜像**: `clickhouse/clickhouse-server:latest`

### 3. 验证集群连接
```bash
kubectl cluster-info
kubectl get nodes
```

## 🚀 部署流程

### Step 1: 环境配置检查

**检查环境配置文件**:
```bash
# 检查目标环境配置
cat deploy/envs/{ENV}.env

# 必需变量:
# - NAMESPACE: 目标命名空间
# - DOCKER_REGISTRY: 镜像仓库地址
# - DOCKER_REPO: 镜像仓库名称
# - IMAGE_TAG: 镜像标签
```

**验证变量设置**:
- `NAMESPACE`: 建议格式 `edge-logs-{env}`
- `IMAGE_TAG`: 确保镜像存在且可访问
- `INGRESS_ENABLED`: 根据需要设置

### Step 2: 执行部署

**方式一: 使用Makefile (推荐)**
```bash
cd deploy
make deploy ENV=dev              # 部署开发环境
make deploy ENV=staging IMAGE_TAG=v1.0.0  # 指定版本部署
```

**方式二: 直接使用脚本**
```bash
./scripts/deploy.sh dev latest
```

### Step 3: 部署状态验证

**检查Pod状态**:
```bash
kubectl get pods -n {NAMESPACE} -o wide
# 预期结果:
# - clickhouse-0: Running
# - edge-logs-apiserver-xxx: Running
# - edge-logs-frontend-xxx: Running
```

**检查服务状态**:
```bash
kubectl get svc -n {NAMESPACE}
# 预期服务:
# - clickhouse: 8123,9000/TCP
# - edge-logs-apiserver: 8080/TCP
# - edge-logs-frontend: 80/TCP
```

**健康检查**:
```bash
# ClickHouse数据库
kubectl exec -n {NAMESPACE} clickhouse-0 -- clickhouse-client --query "SELECT 1"

# APIServer (需要真实镜像)
kubectl exec -n {NAMESPACE} deployment/edge-logs-apiserver -- wget -qO- http://localhost:8080/api/v1alpha1/health

# 服务间连通性
kubectl exec -n {NAMESPACE} deployment/edge-logs-frontend -- curl -I http://edge-logs-apiserver.{NAMESPACE}.svc.cluster.local:8080
```

### Step 4: 部署验证清单

- [ ] 所有Pod状态为Running
- [ ] 所有Service正确创建
- [ ] ClickHouse数据库连接正常
- [ ] APIServer健康检查通过
- [ ] Frontend可访问APIServer
- [ ] ConfigMap正确创建和挂载
- [ ] RBAC权限配置正确

## 🔧 故障排查

### 常见问题及解决方案

#### 1. 镜像拉取失败
**问题**: `ErrImagePull` 或 `ImagePullBackOff`
```bash
# 检查pod详情
kubectl describe pod {POD_NAME} -n {NAMESPACE}

# 解决方案:
# - 确认镜像存在: docker pull {IMAGE}
# - 检查镜像仓库认证
# - 验证网络连通性
```

#### 2. Pod启动失败
**问题**: Pod状态为`CrashLoopBackOff`
```bash
# 查看日志
kubectl logs {POD_NAME} -n {NAMESPACE} -f

# 查看事件
kubectl get events -n {NAMESPACE} --sort-by='.lastTimestamp'

# 解决方案:
# - 检查配置文件语法
# - 验证环境变量设置
# - 确认依赖服务可用
```

#### 3. 服务连通性问题
**问题**: 服务间无法通信
```bash
# DNS解析测试
kubectl exec -n {NAMESPACE} {POD_NAME} -- nslookup {SERVICE}.{NAMESPACE}.svc.cluster.local

# 网络连通性测试
kubectl exec -n {NAMESPACE} {POD_NAME} -- telnet {SERVICE} {PORT}

# 解决方案:
# - 检查Service选择器标签
# - 验证端口映射正确性
# - 确认网络策略设置
```

### 恢复操作

#### 快速重启
```bash
# 重启特定组件
kubectl rollout restart deployment/edge-logs-apiserver -n {NAMESPACE}
kubectl rollout restart deployment/edge-logs-frontend -n {NAMESPACE}
kubectl rollout restart statefulset/clickhouse -n {NAMESPACE}
```

#### 完全重新部署
```bash
# 删除命名空间 (谨慎操作)
kubectl delete namespace {NAMESPACE}

# 重新部署
make deploy ENV={ENV}
```

#### 回滚操作
```bash
# 查看部署历史
kubectl rollout history deployment/edge-logs-apiserver -n {NAMESPACE}

# 回滚到上一版本
kubectl rollout undo deployment/edge-logs-apiserver -n {NAMESPACE}

# 回滚到指定版本
kubectl rollout undo deployment/edge-logs-apiserver --to-revision=2 -n {NAMESPACE}
```

## 📊 监控和维护

### 日常检查命令
```bash
# 快速状态检查
make status ENV={ENV}

# 查看日志
make logs ENV={ENV} COMPONENT=apiserver
make logs ENV={ENV} COMPONENT=frontend

# 资源使用情况
kubectl top pods -n {NAMESPACE}
```

### 性能监控
```bash
# Pod资源使用
kubectl top pods -n {NAMESPACE}

# 节点资源使用
kubectl top nodes

# 存储使用 (ClickHouse)
kubectl exec -n {NAMESPACE} clickhouse-0 -- du -sh /var/lib/clickhouse
```

## 🔒 安全注意事项

1. **生产环境部署**: 需要额外确认步骤
2. **镜像安全**: 使用已扫描的镜像
3. **网络策略**: 根据需要配置网络隔离
4. **RBAC权限**: 最小权限原则
5. **敏感信息**: 使用Secret而非ConfigMap

## 📚 参考文档

- [部署脚本](./scripts/deploy.sh)
- [环境配置](./envs/)
- [K8s清单](./k8s/)
- [故障排查手册](./README.md#故障排查)

---

**更新记录**:
- 2026-01-11: 初始版本，基于实际部署经验编写
- 包含完整的部署流程、验证步骤和故障排查指南