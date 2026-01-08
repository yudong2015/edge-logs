---
stepsComplete: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments: []
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '2026-01-08'
project_name: 'edge-logs'
user_name: 'Neov'
date: '2026-01-08'
---

# 架构决策文档

## 1. 概述

### 1.1 项目愿景

构建一个基于 ClickHouse 的云原生日志平台，专注于边缘计算场景，提供：
- 优秀的日志查询 API
- 云原生日志采集支持（使用开源组件）
- 简化的架构设计

### 1.2 设计原则

- **查询能力为主** - 提供日志查询 API
- **采集链路用开源** - iLogtail 直接写 ClickHouse
- **简化架构** - 单一 API 服务组件
- **数据隔离** - dataset 作为独立字段，支持多数据源逻辑隔离

## 2. 技术栈（与 edge-apiserver 保持一致）

| 组件 | 版本 | 用途 |
|------|------|------|
| **语言** | Go 1.23 | - |
| **Web 框架** | go-restful/v3 | HTTP API |
| **日志框架** | klog/v2 | 结构化日志 |
| **启动参数** | cobra | 命令行 |
| **ClickHouse 驱动** | clickhouse-go/v2 | 数据访问 |
| **K8s 客户端** | client-go v0.31.2 | 元数据关联 |

**外部开源组件：**

| 组件 | 用途 |
|------|------|
| iLogtail | 日志采集（直接写 ClickHouse） |
| ClickHouse | 日志存储 |
| KubeEdge/OpenYurt | 边缘自治（可选） |

## 3. 项目结构

```
edge-logs/
├── cmd/
│   └── apiserver/           # API 服务器 (go-restful)
│       └── main.go
├── config/
│   ├── config.go
│   └── config.yaml
├── pkg/
│   ├── apiserver/           # go-restful 容器
│   │   └── apiserver.go
│   ├── oapis/               # API handlers (go-restful)
│   │   └── log/v1alpha1/    # 日志查询 API
│   ├── model/
│   │   ├── request/         # API 请求模型
│   │   ├── response/        # API 响应模型
│   │   └── clickhouse/      # ClickHouse 数据模型
│   ├── repository/
│   │   └── clickhouse/      # ClickHouse 数据访问
│   ├── service/
│   │   ├── query/           # 日志查询服务
│   │   └── enrichment/      # 元数据关联（从 K8s API）
│   ├── middleware/
│   │   ├── ratelimit.go
│   │   └── logging.go
│   ├── filters/             # go-restful 过滤器
│   │   └── requestinfo.go
│   ├── config/              # 配置管理
│   ├── constants/           # 常量定义
│   └── response/            # API 响应
├── deploy/
│   ├── apiserver/
│   │   └── Dockerfile
│   └── helm/
│       └── charts/
├── hack/
│   ├── boilerplate.go.txt
│   └── docker_build.sh
├── sqlscripts/
│   └── clickhouse/
│       ├── 01_tables.sql
│       └── 02_indexes.sql
├── test/
│   ├── e2e/
│   └── integration/
├── .github/
│   └── workflows/
│       ├── lint.yml
│       ├── test.yml
│       ├── build.yml
│       └── security.yml
├── .golangci.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## 4. 架构设计

### 4.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        整体架构                                  │
│                                                                  │
│  边缘节点                         云端                          │
│  ┌──────────────┐          ┌─────────────────────────────────┐  │
│  │  iLogtail    │─────────▶│         ClickHouse              │  │
│  │  (开源采集)   │  直接写入│      (开源存储)                  │  │
│  └──────────────┘          └──────────────┬──────────────────┘  │
│                                           ▲                     │
│                                           │ 查询                │
│                                           ▼                     │
│                                  ┌─────────────────────────────────┐  │
│                                  │      edge-logs API Server       │  │
│                                  │         (查询能力)               │  │
│                                  │         go-restful/v3          │  │
│                                  └─────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 ClickHouse Schema 设计

**决策：** 基于时间分区的 MergeTree 引擎

**表结构：**

```sql
-- 主日志表 (iLogtail 直接写入)
CREATE TABLE IF NOT EXISTS logs (
    -- 时间和数据隔离
    timestamp          DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    dataset            LowCardinality(String) CODEC(ZSTD(1)),

    -- 日志内容
    content            String CODEC(ZSTD(1)),
    severity           LowCardinality(String) CODEC(ZSTD(1)),

    -- 容器信息
    container_id       String CODEC(ZSTD(1)),
    container_name     LowCardinality(String) CODEC(ZSTD(1)),
    pid                String CODEC(ZSTD(1)),

    -- 主机信息
    host_ip            LowCardinality(String) CODEC(ZSTD(1)),
    host_name          LowCardinality(String) CODEC(ZSTD(1)),

    -- K8s 元数据
    k8s_namespace_name LowCardinality(String) CODEC(ZSTD(1)),
    k8s_pod_name       LowCardinality(String) CODEC(ZSTD(1)),
    k8s_pod_uid        String CODEC(ZSTD(1)),
    k8s_node_name      LowCardinality(String) CODEC(ZSTD(1)),

    -- 标签字段 (cluster、region 等分析维度)
    tags               Map(String, String) CODEC(ZSTD(1)),

    -- 全文搜索索引
    INDEX idx_content content TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
    -- tags 索引 (加速 cluster 等标签查询)
    INDEX idx_tags tags TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY (dataset, toDate(timestamp))
ORDER BY (dataset, host_ip, timestamp)
TTL timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- 集群表支持 (可选，用于 ClickHouse 集群)
CREATE TABLE IF NOT EXISTS logs_distributed
ENGINE = Distributed('{cluster}', 'default', 'logs', rand());
```

**设计说明（参�� APO 生产经验 + dataset 隔离）：**

| 设计点 | 说明 |
|--------|------|
| **dataset 作为 Key** | 路由隔离维度，放在 URL 路径中 |
| **cluster 作为标签** | 分析维度，放在 tags['cluster'] 中，架构不绑死 |
| **显式字段** | K8s 元数据使用独立 LowCardinality 列，比 Map 查询性能高 10x+ |
| **CODEC 压缩** | Delta+ZSTD 压缩，存储节省 70%+ |
| **ORDER BY** | `(dataset, host_ip, timestamp)` - 按数据集+主机分组，时间范围查询最优 |
| **PARTITION BY** | `(dataset, date)` - 支持按数据集级别删除/管理 |
| **tokenbf_v1** | 32768 粒度，生产验证的全文搜索配置 |
| **bloom_filter 索引** | 加速 tags['cluster'] 等标签查询 |

**数据结构示例：**

```json
{
  "dataset": "edge-prod-traffic",
  "timestamp": 1736312345,
  "content": "starting kubelet...",
  "severity": "info",
  "container_name": "kubelet",
  "host_name": "node-23",
  "k8s_node_name": "node-23",
  "tags": {
    "cluster": "hz-edge-01",
    "region": "cn-hangzhou",
    "source": "kubelet",
    "env": "prod"
  }
}
```

**使用场景：**

| 场景 | 查询方式 |
|------|----------|
| 按 dataset 路由 | `GET /apis/.../logdatasets/edge-prod-traffic/logs` |
| 按 cluster 分析 | `WHERE tags['cluster'] = 'hz-edge-01'` |
| 按 node 分析 | `WHERE tags['node'] = 'node-23'` |
| 组合分析 | `WHERE tags['cluster'] = ? AND tags['region'] = ?` |

**iLogtail 字段映射：**

| ClickHouse 列 | iLogtail 字段 | 来源 |
|---------------|---------------|------|
| timestamp | timestamp | 日志时间 |
| **dataset** | **环境变量 LOG_DATASET** | **数据集标识（必需）** |
| content | body | 日志内容 |
| severity | level | 日志级别 |
| container_id | _container_id_ | 容器ID |
| container_name | _container_name_ | 容器名 |
| pid | _pid_ | 进程ID |
| host_ip | _host_ip_ | 主机IP |
| host_name | _host_name_ | 主机名 |
| k8s_namespace_name | k8s.namespace.name | 命名空间 |
| k8s_pod_name | k8s.pod.name | Pod名 |
| k8s_pod_uid | k8s.pod.uid | Pod UID |
| k8s_node_name | k8s.node.name | 节点名 |
| **tags['cluster']** | **环境变量 CLUSTER_NAME** | **集群名称（标签）** |
| tags | 其他自定义标签 | Map 存储 |

**iLogtail 配置示例：**

```yaml
# iLogtail DaemonSet 配置
env:
  # dataset: 路由隔离 Key
  - name: LOG_DATASET
    valueFrom:
      configMapKeyRef:
        name: log-config
        key: dataset
  # cluster: 分析维度标签
  - name: CLUSTER_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.labels['topology.kubernetes.io/cluster']
  # 其他标签
  - name: REGION_NAME
    value: "cn-hangzhou"
```

### 4.3 API 设计

**决策：** 使用 K8s API Aggregation 模式，通过 APIServer 统一接入

**API Group：** `log.theriseunion.io`
**版本：** `v1alpha1`

**API 端点：**

| 分类 | 路径 | 用途 |
|------|------|------|
| **查询** | GET /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs | 日志查询 |
| **聚合** | POST /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/aggregations | 日志聚合 |
| **健康** | GET /healthz | 健康检查（K8s 标准）|
| **指标** | GET /metrics | Prometheus 指标 |

**查询参数设计：**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| dataset | path | 是 | 数据集标识（资源名） |
| start_time | query | 是 | 查询起始时间 |
| end_time | query | 是 | 查询结束时间 |
| namespace | query | 否 | 按命名空间过滤 |
| pod_name | query | 否 | 按 Pod 名过滤 |
| filter | query | 否 | 日志内容关键词 |
| limit | query | 否 | 返回条数限制 |

**go-restful WebService 结构：**

```go
// 日志查询 API (K8s Aggregation 模式)
ws := new(restful.WebService)
ws.Path("/apis/log.theriseunion.io/v1alpha1").
    Consumes(restful.MIME_JSON).
    Produces(restful.MIME_JSON)

// /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/logs
ws.Route(ws.GET("/logdatasets/{dataset}/logs").To(queryHandler).
    Doc("Query logs from a dataset").
    Param(ws.PathParameter("dataset", "Dataset identifier")).
    Param(ws.QueryParameter("start_time", "Start timestamp (required)").DataType("dateTime")).
    Param(ws.QueryParameter("end_time", "End timestamp (required)").DataType("dateTime")).
    Param(ws.QueryParameter("namespace", "Filter by namespace")).
    Param(ws.QueryParameter("pod_name", "Filter by pod name")).
    Param(ws.QueryParameter("filter", "Log content filter")).
    Param(ws.QueryParameter("limit", "Max records to return")))

// /apis/log.theriseunion.io/v1alpha1/logdatasets/{dataset}/aggregations
ws.Route(ws.POST("/logdatasets/{dataset}/aggregations").To(aggregateHandler).
    Doc("Aggregate logs by dimensions").
    Param(ws.PathParameter("dataset", "Dataset identifier")))

container.Add(ws)
```

**APIServer Aggregation 配置：**

```yaml
# /etc/kubernetes/manifests/kube-apiserver.yaml
--enable-aggregator-routing=true
```

```yaml
# APIService 注册
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: v1alpha1.log.theriseunion.io
spec:
  group: log.theriseunion.io
  version: v1alpha1
  service:
    namespace: edge-logs
    name: apiserver
    port: 443
  caBundle: ${CA_BUNDLE}
  priority: 100
  groupPriorityMinimum: 1000
  versionPriority: 15
```

**路由示例：**

```bash
# 通过 K8s APIServer 统一访问
kubectl get --raw="/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/logs?start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T01:00:00Z"

# 或直接访问（聚合层会转发）
curl -k https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/edge-cn/logs?namespace=default&pod_name=app-xxx

# 聚合查询
curl -k -X POST https://apiserver:443/apis/log.theriseunion.io/v1alpha1/logdatasets/prod-cluster/aggregations \
  -H "Content-Type: application/json" \
  -d '{"group_by": ["severity", "namespace"], "aggregations": {"count": "*"}}'
```

### 4.4 元数据关联策略

**决策：** K8s 元数据已作为显式列存储，无需关联。Labels/Annotations 按需关联。

```
┌─────────────────────────────────────────────────────────────────┐
│                     查询流程                                     │
│                                                                  │
│  1. 客户端请求: GET /apis/log.theriseunion.io/v1alpha1/...     │
│                           │                                    │
│                           ▼                                    │
│  2. ClickHouse 查询 (所有元数据已存在，直接查��)                   │
│     ┌──────────────────────────────────────┐                   │
│     │ SELECT timestamp, content,          │                   │
│     │   k8s_namespace_name, k8s_pod_name, │                   │
│     │   k8s_node_name, severity, tags     │                   │
│     │ FROM logs                           │                   │
│     │ WHERE dataset = ?                   │                   │
│     │   AND timestamp BETWEEN ? AND ?     │                   │
│     │   AND k8s_namespace_name = ?        │                   │
│     └──────────────────────────────────────┘                   │
│                           │                                    │
│                           ▼                                    │
│  3. (可选) 如需 Pod Labels/Annotations，从 K8s API 关联          │
│     ┌──────────────────────────────────────┐                   │
│     │ GET /api/v1/namespaces/{ns}/pods/{name}│                 │
│     │ → labels.app, env, team 等           │                   │
│     └──────────────────────────────────────┘                   │
│                           │                                    │
│                           ▼                                    │
│  4. 返回结果                                                     │
└─────────────────────────────────────────────────────────────────┘
```

**元数据可用性：**

| 元数据类型 | 获取方式 | 来源 |
|-----------|---------|------|
| dataset | ClickHouse 显式列 | iLogtail 环境变量 |
| timestamp | ClickHouse 显式列 | iLogtail |
| content | ClickHouse 显式列 | iLogtail |
| severity | ClickHouse 显式列 | iLogtail |
| container_id/name | ClickHouse 显式列 | iLogtail (CRI) |
| host_ip/name | ClickHouse 显式列 | iLogtail |
| k8s_namespace_name | ClickHouse 显式列 | iLogtail (CRI) |
| k8s_pod_name | ClickHouse 显式列 | iLogtail (CRI) |
| k8s_pod_uid | ClickHouse 显式列 | iLogtail (CRI) |
| k8s_node_name | ClickHouse 显式列 | iLogtail (CRI) |
| **Pod Labels** | K8s API 关联（可选） | K8s API |
| **Pod Annotations** | K8s API 关联（可选） | K8s API |
| tags | ClickHouse Map 列 | iLogtail 自定义 |

**设计优势：**
- 95% 的查询无需访问 K8s API，直接从 ClickHouse 获取
- LowCardinality 列 + CODEC，存储和查询性能最优
- 按需关联 Labels/Annotations，不影响核心链路

## 5. 边缘场景支持

### 5.1 iLogtail 元数据获取

**iLogtail 有两种元数据获取方式：**

```
┌─────────────────────────────────────────────────────────────────┐
│                        iLogtail                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  方式 1: CRI 接口 (无需 apiserver) ✅ 边缘场景友好              │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  containerd / CRI-O → 容器沙箱元数据                      │   │
│  │  - 容器名称、ID                                           │   │
│  │  - Pod 名称、UID                                          │   │
│  │  - Namespace 名称                                         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  方式 2: Kubernetes Metadata Plugin (需要 apiserver)             │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  - 获取 Labels、Annotations                               │   │
│  │  - 边缘断网时不可用                                        │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 边缘框架集成

**推荐：KubeEdge / OpenYurt**

| 框架 | 维护者 | CNCF 状态 | 边缘自治能力 |
|------|--------|-----------|-------------|
| KubeEdge | 阿里/社区 | 沙箱项目 | ✅ EdgeHub 断网自治 |
| OpenYurt | 阿里 | CNCF | ✅ Raven 跨网络 |

**边缘自治由框架提供，edge-logs 无需额外处理。**

## 6. 实施优先级

### 6.1 首个实施优先级

1. 初始化 Go 模块和项目结构
2. 创建 ClickHouse 表结构
3. 实现 ClickHouse Repository 层
4. 实现日志查询 Service
5. 实现查询 API Handler（go-restful）
6. 实现元数据关联 Service（从 K8s API）
7. 添加 Prometheus 指标
8. 创建 Helm Chart

### 6.2 快速启动

```bash
# 初始化项目
mkdir -p cmd/apiserver pkg hack config
go mod init github.com/outpostos/edge-logs

# 添加依赖 (与 edge-apiserver 保持一致)
go get github.com/emicklei/go-restful/v3
go get github.com/emicklei/go-restful-openapi/v2
go get github.com/spf13/cobra
go get k8s.io/klog/v2
go get github.com/ClickHouse/clickhouse-go/v2
go get k8s.io/client-go
```

## 7. 代码规范

### 7.1 日志记录（klog/v2）

```go
import "k8s.io/klog/v2"

// 基础日志
klog.Info("Starting log query")
klog.Warning("High memory usage")
klog.Error("Failed to connect to ClickHouse")

// 结构化日志
klog.InfoS("Log query started",
    "pod_name", podName,
    "namespace", namespace,
    "start_time", startTime)

// 详细日志 (V级别)
klog.V(2).Info("Detailed debug info")
```

### 7.2 API 响应格式

```go
type Response struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

type LogQueryResponse struct {
    Items      []LogEntry `json:"items"`
    Total      int64      `json:"total"`
    Page       int        `json:"page"`
    Limit      int        `json:"limit"`
    TotalPages int        `json:"total_pages"`
}
```

---

**架构状态：** 已准备好实施 ✅

**文档维护：** 在实施过程中做出重大技术决策时更新本架构。
