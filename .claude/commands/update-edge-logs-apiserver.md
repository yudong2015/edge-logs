# update-edge-logs-apiserver

重新构建并更新 edge-logs-apiserver 到测试环境（119.8.182.199）。

## 执行步骤

### Step 1: 同步代码到测试节点

```bash
rsync -avrP --delete ~/workspace/theriseunion/edge-logs root@119.8.182.199:/root/
```

### Step 2: 在主节点生成镜像 tag 并构建 apiserver 镜像

生成 8 位随机 tag，然后构建 apiserver 镜像：

```bash
ssh root@119.8.182.199 "
  cd /root/edge-logs && \
  NEW_TAG=\$(cat /dev/urandom | tr -dc 'a-z0-9' | head -c 8) && \
  echo \"Building with tag: \$NEW_TAG\" && \
  docker build -t quanzhenglong.com/edge/logs-apiserver:\$NEW_TAG -f deploy/apiserver/Dockerfile . && \
  echo \"NEW_TAG=\$NEW_TAG\"
"
```

记录输出中的 `NEW_TAG` 值，后续步骤使用。

### Step 3: 更新 deployment 镜像

使用 Step 2 中获得的 `NEW_TAG` 执行更新：

```bash
kubectl set image deployment/edge-logs-apiserver apiserver=quanzhenglong.com/edge/logs-apiserver:[NEW_TAG] -n logging-system
```

将 `[NEW_TAG]` 替换为 Step 2 输出的实际 tag 值。

### Step 4: 验证部署

```bash
# 检查 pod 状态
kubectl get pods -n logging-system | grep apiserver

# 验证新镜像
kubectl get pod -n logging-system -l app=logs-apiserver -o jsonpath='{.items[0].spec.containers[0].image}'

# 测试 API
kubectl port-forward -n logging-system svc/edge-logs-apiserver 8080:8080 &
PF_PID=$!
sleep 3
curl -s http://localhost:8080/apis/log.theriseunion.io/v1alpha1/datasets | jq .
kill $PF_PID 2>/dev/null
```

## 注意事项

- 确保本地 `~/.ssh/config` 或 ssh key 已配置好对 `119.8.182.199` 的免密登录
- Deployment 会自动滚动更新，无需手动删除旧 pod
- 新 pod 启动时会等待 ClickHouse 就绪（initContainer）
