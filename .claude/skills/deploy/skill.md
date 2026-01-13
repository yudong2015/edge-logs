---
description: K8s Deployment Agent
trigger:
  - deploy edge-logs
  - helm install
  - kubernetes deployment
---

# Edge-Logs K8s Deployment Skill

云原生部署专家，以**自动化模式**运行，持续推进服务部署到目标 K8s 上，直到遇到 HALT 条件。

## 部署理念

### 不可变基础设施
- 不允许本地编译镜像
- 通过 GitHub Actions 构建镜像，每个镜像和代码版本关联
- 通过 Secret、Var 来指定镜像版本和地址

### 可复现、可回滚的部署流程
- 通过 Helm Chart 的版本来管理部署
- Chart 指定服务之间的关系，镜像版本通过配置来确定

### 应该修改流程，而不是结果
- 不允许修改结果，应该修改流程
- 如果有问题可以尝试修复，如果有问题终止部署

## 组件架构

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  Frontend   │────▶│  APIServer   │────▶│ ClickHouse │
└─────────────┘     └──────────────┘     └────────────┘
                                               ▲
┌─────────────┐     ┌──────────────┐           │
│  iLogtail   │────▶│OTEL Collector│───────────┘
│ (DaemonSet) │     └──────────────┘
└─────────────┘
```

## 组件清单

| 组件 | 类型 | 镜像 | 用途 |
|------|------|------|------|
| edge-logs-apiserver | Deployment | quanzhenglong.com/edge/logs-apiserver:main | 日志查询 API |
| edge-logs-frontend | Deployment | quanzhenglong.com/edge/logs-frontend:main | Web UI |
| clickhouse | StatefulSet | quanzhenglong.com/edge/clickhouse-server:24.3 | 日志存储 |
| otel-collector | Deployment | quanzhenglong.com/edge/opentelemetry-collector-contrib:0.98.0 | 日志转发 |
| ilogtail | DaemonSet | quanzhenglong.com/edge/ilogtail:2.0.4 | 日志采集 |

## 部署顺序

**依赖关系决定部署顺序：**

1. **ClickHouse** - 数据库，无依赖，必须先启动
2. **OTEL Collector** - 依赖 ClickHouse 端口 9000
3. **iLogtail** - 依赖 OTEL Collector 端口 4317
4. **APIServer** - 依赖 ClickHouse 端口 9000
5. **Frontend** - 依赖 APIServer 端口 8080

## 目标集群

- 连接方式: `ssh hw101`
- 镜像仓库: `quanzhenglong.com/edge/`

## 关键集成

- APIServer 注册为 `/apis/log.theriseunion.io/v1alpha1/*`
- 验证: `kubectl get apiservice | grep log.theriseunion.io`
- ClickHouse 服务发现: `clickhouse.${NAMESPACE}.svc.cluster.local`

## 镜像管理

### 公共镜像迁移到私有仓库

使用 skopeo 将镜像拷贝到 `quanzhenglong.com/edge/`:

```bash
skopeo copy --all \
  docker://docker.io/clickhouse/clickhouse-server:24.3 \
  docker://quanzhenglong.com/edge/clickhouse-server:24.3

skopeo copy --all \
  docker://docker.io/otel/opentelemetry-collector-contrib:0.98.0 \
  docker://quanzhenglong.com/edge/opentelemetry-collector-contrib:0.98.0

skopeo copy --all \
  docker://sls-opensource-registry.cn-shanghai.cr.aliyuncs.com/ilogtail-community-edition/ilogtail:2.0.4 \
  docker://quanzhenglong.com/edge/ilogtail:2.0.4
```

### 专有镜像通过 GitHub Actions 构建

- **触发**: push 到 main/develop/feature 分支或创建 tag
- **输出**: `quanzhenglong.com/edge/logs-apiserver:${branch}`
- **架构**: linux/amd64, linux/arm64

**环境变量配置** (GitHub 仓库设置):
- `DOCKER_REGISTRY`: quanzhenglong.com (变量)
- `DOCKER_REPO`: edge (变量)
- `DOCKER_USERNAME`: edge_admin (变量)
- `DOCKER_PASSWORD`: ****** (密钥)

**镜像标签规则**:
- `main`: main 分支最新
- `develop`: develop 分支
- `v1.0.0`: tag 版本
- `main-abc1234`: 分支名-短 SHA


| 规则 | 触发条件 | 生成的标签示例 |
|---------------------------------------------|----------------------------|------------------------|
| type=semver,pattern={{version}}              | 推送 git tag（如 v1.2.3）  | 1.2.3               |
| type=semver,pattern={{major}}.{{minor}}      | 推送 git tag（如 v1.2.3）  | 1.2                 |
| type=ref,event=branch,enable=...             | 分支名以 release- 开头     | release-1.0         |
| type=raw,value={{branch}}-{{sha}},enable=... | 分支名以 release- 开头     | release-1.0-a3da933 |
| type=raw,value=main,enable=...               | 手动触发 workflow_dispatch | main                |
| type=raw,value=main-{{sha}},enable=...       | 在 main 分支上             | main-a3da933        |

实际场景举例：

  1. 打 tag v1.2.3 时 → 生成 1.2.3 和 1.2 两个镜像标签
  2. 在 release-1.0 分支推送 → 生成 release-1.0 和 release-1.0-{sha} 标签
  3. 手动触发工作流 → 生成 main 标签
  4. 在 main 分支推送 → 生成 main-{sha} 标签

  enable 参数控制该规则是否生效，只有条件为 true 时才会生成对应标签。


## 健康检查端点

| 组件 | 健康检查 URL | 预期响应 |
|------|--------------|----------|
| ClickHouse | `GET http://clickhouse:8123/ping` | `Ok.` |
| OTEL Collector | `GET http://otel-collector:13133/` | 200 OK |
| APIServer | `GET http://edge-logs-apiserver:8080/api/v1alpha1/health` | `{"status":"ok"}` |
| Frontend | `GET http://edge-logs-frontend/healthz` | 200 OK |

## 目录结构

```
# 部署资源 (项目常规位置)
deploy/
├── helm/                 # Helm Chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
├── scripts/
│   ├── deploy.sh         # 部署脚本
│   ├── rollback.sh       # 回滚脚本
│   ├── migrate-images.sh # 镜像迁移
│   └── logs.sh           # 日志查看

# 检查脚本 (skill 的一部分)
.claude/skills/deploy/
├── skill.md              # 本文档
└── scripts/
    ├── verify-images.sh  # 镜像验证
    ├── status.sh         # 状态检查
    └── acceptance-test.sh # 验收测试
```

## 部署命令

### 方式 1: Helm 部署 (推荐)

```bash
# 使用默认仓库部署
helm install edge-logs ./deploy/helm \
  -n edge-logs --create-namespace

# 使用私有仓库部署
helm install edge-logs ./deploy/helm \
  -n edge-logs --create-namespace \
  --set global.imageRegistry=my-registry.com/edge

# 升级部署
helm upgrade edge-logs ./deploy/helm -n edge-logs

# 卸载
helm uninstall edge-logs -n edge-logs
```

### 方式 2: 脚本部署

```bash
cd deploy/scripts

# 部署
./deploy.sh

# 回滚
./rollback.sh
```

## 检查命令

```bash
cd .claude/skills/deploy/scripts

# 验证镜像
./verify-images.sh

# 检查状态
./status.sh

# 验收测试
./acceptance-test.sh
```

## 私有仓库配置

### values.yaml 配置方式

```yaml
# 全局配置 (推荐)
global:
  imageRegistry: "my-registry.com/edge"

# 单独配置某个组件
apiserver:
  image:
    registry: "my-registry.com/custom"
    repository: logs-apiserver
    tag: "v1.0.0"
```

### 镜像生成逻辑

```
最终镜像 = {registry | global.imageRegistry}/{repository}:{tag}
```

## 验证部署

```bash
# 1. 检查所有 Pod 状态
kubectl get pods -n edge-logs

# 2. 检查服务
kubectl get svc -n edge-logs

# 3. 测试 API
kubectl port-forward svc/edge-logs-apiserver 8080:8080 -n edge-logs &
curl http://localhost:8080/api/v1alpha1/logs/query?limit=3

# 4. 测试前端
kubectl port-forward svc/edge-logs-frontend 8081:80 -n edge-logs &
curl http://localhost:8081/healthz
```

## HALT 条件

部署过程中遇到以下情况应停止并报告：

- `HALT: 集群连接失败`
- `HALT: 镜像拉取失败`
- `HALT: Pod 启动失败` (连续失败3次)
- `HALT: 健康检查失败` (超时60秒)

## 常见问题

### ClickHouse 启动失败
- 检查 PVC 是否创建成功
- 检查存储类是否可用

### OTEL Collector 无法连接 ClickHouse
- 确认 ClickHouse 已 Ready
- 检查 Service 名称是否正确: `clickhouse:9000`

### iLogtail 无日志输出
- 检查 OTEL Collector 是否正常
- 检查 RBAC 权限是否正确

### APIServer 查询超时
- 检查 ClickHouse 连接字符串
- 检查网络策略是否阻止访问
