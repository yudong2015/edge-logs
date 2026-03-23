# reinstall-edge-logs

重新构建并部署 edge-logs 到测试环境（119.8.182.199）。

## 执行步骤

### Step 1: 同步代码到测试节点

```bash
rsync -avrP --delete ~/workspace/theriseunion/edge-logs root@119.8.182.199:/root/
```

### Step 2: 在主节点生成镜像 tag 并构建镜像

生成 8 位随机 tag，然后构建 apiserver 和 frontend 镜像：

```bash
ssh root@119.8.182.199 "
  cd /root/edge-logs && \
  NEW_TAG=\$(cat /dev/urandom | tr -dc 'a-z0-9' | head -c 8) && \
  echo \"Building with tag: \$NEW_TAG\" && \
  docker build -t quanzhenglong.com/edge/logs-apiserver:\$NEW_TAG -f deploy/apiserver/Dockerfile . && \
  cd frontend && docker build -t quanzhenglong.com/edge/logs-frontend:\$NEW_TAG . && \
  echo \"NEW_TAG=\$NEW_TAG\"
"
```

记录输出中的 `NEW_TAG` 值，后续步骤使用。

### Step 3: 清理现有 helm release（如果存在）

先检查是否存在 `edge-logs` release：

```bash
helm list -n logging-system | grep edge-logs
```

如果存在，执行清理：

```bash
# 卸载 helm release
helm uninstall edge-logs -n logging-system

# 按 label 自动删除 clickhouse 的 PVC
kubectl delete pvc -n logging-system -l app.kubernetes.io/instance=edge-logs
```

### Step 3.5: 安装依赖 CRDs（首次部署或 CRD 被清理时需要）

edge-logs 依赖 edge-apiserver 的 CRDs，需要先安装：

```bash
# 安装 IAM CRDs
ls ~/workspace/theriseunion/edge-apiserver/config/crd/bases/ | grep "iam\.theriseunion\.io" | \
  xargs -I {} kubectl apply -f ~/workspace/theriseunion/edge-apiserver/config/crd/bases/{}

# 安装 scope CRDs
ls ~/workspace/theriseunion/edge-apiserver/config/crd/bases/ | grep "scope\.theriseunion\.io" | \
  xargs -I {} kubectl apply -f ~/workspace/theriseunion/edge-apiserver/config/crd/bases/{}
```

### Step 4: 部署 edge-logs

使用 Step 2 中获得的 `NEW_TAG` 执行部署：

```bash
ssh root@119.8.182.199 "
  cd /root/edge-logs/deploy/helm && \
  helm install edge-logs -n logging-system \
    --set apiserver.image.tag=[NEW_TAG] \
    --set frontend.image.tag=[NEW_TAG] \
    .
"
```

将 `[NEW_TAG]` 替换为 Step 2 输出的实际 tag 值。

### Step 4.5: 暴露前端服务

```bash
kubectl apply -f /root/frontend-svc.yaml
```

### Step 5: 验证部署

```bash
# 检查 pod 状态
kubectl get pods -n logging-system

# 检查 helm release 状态
helm list -n logging-system

# 查看 pod 日志（如有异常）
kubectl logs -n logging-system -l app=logs-apiserver --tail=50
kubectl logs -n logging-system -l app=logs-frontend --tail=50
```

## 注意事项

- 确保本地 `~/.ssh/config` 或 ssh key 已配置好对 `119.8.182.199` 的免密登录
- PVC 按 `app.kubernetes.io/instance=edge-logs` label 删除，会自动匹配 clickhouse 的 PVC
