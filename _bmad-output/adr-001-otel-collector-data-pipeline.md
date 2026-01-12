---
title: "ADR-001: 采用 OTEL Collector 作为日志数据管道中间件"
date: "2026-01-12"
status: "accepted"
deciders: ["Neov"]
reference: "APO 项目标准方案"
---

# ADR-001: 采用 OTEL Collector 作为日志数据管道中间件

## 状态

**已接受** (Accepted)

## 背景

当前 edge-logs 项目的日志采集架构采用 iLogtail 直接通过 HTTP 接口写入 ClickHouse 的方案：

```
当前架构:
iLogtail (DaemonSet) --[HTTP POST]--> ClickHouse (8123 端口)
```

这种方案在实际运行中遇到了以下问题：

1. **数据格式不兼容**: iLogtail 设计为发送 OpenTelemetry 格式数据，直接发送到 ClickHouse 需要复杂的格式转换
2. **时间戳处理困难**: 当前配置使用固定时间戳作为临时方案，无法正确处理动态时间戳
3. **元数据采集不完整**: 缺少完整的 Kubernetes 元数据（container_name, pod_name, namespace 等）
4. **缺少中间层处理能力**: 没有数据过滤、转换、聚合的能力
5. **与行业标准不一致**: APO 等成熟项目均采用 OTEL Collector 作为中间件

## 决策

**采用 iLogtail → OTEL Collector → ClickHouse 的三层架构**

```
目标架构:
┌─────────────┐        ┌────────────────┐        ┌──────────────┐
│  iLogtail   │───────►│ OTEL Collector │───────►│  ClickHouse  │
│ (DaemonSet) │  OTLP  │  (Deployment)  │   TCP  │ (StatefulSet)│
└─────────────┘  gRPC  └────────────────┘  9000  └──────────────┘
                4317
```

### 关键设计决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| **传输协议** | gRPC (4317) | APO 标准方案，性能优于 HTTP |
| **表结构** | 修改现有 logs 表 | 保持 API 层兼容性，避免数据迁移 |
| **OTEL Collector 部署** | Deployment (单副本) | 足够应对当前规模，易于扩展 |

## 详细设计

### 1. 数据流架构

```
┌──────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ 应用层 Pod (所有命名空间)                                    │ │
│  │ 生成日志到 /var/log/containers/*.log                        │ │
│  └────────────────────────┬────────────────────────────────────┘ │
│                           │                                       │
│                     [文件采集]                                    │
│                           ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ iLogtail DaemonSet                                          │ │
│  │ - 采集: /var/log/containers/*.log                           │ │
│  │ - 处理: JSON 解析, K8s 元数据采集                           │ │
│  │ - 输出: OTLP gRPC (端口 4317)                               │ │
│  └────────────────────────┬────────────────────────────────────┘ │
│                           │                                       │
│                    [OTLP gRPC]                                    │
│                           ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ OTEL Collector Deployment                                   │ │
│  │ - 接收: OTLP gRPC (4317), HTTP (4318)                       │ │
│  │ - 处理: Transform (字段映射), Batch (批处理)                │ │
│  │ - 输出: ClickHouse Exporter (TCP 9000)                      │ │
│  └────────────────────────┬────────────────────────────────────┘ │
│                           │                                       │
│                   [TCP 原生协议]                                  │
│                           ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ ClickHouse StatefulSet                                      │ │
│  │ - 数据库: edge_logs                                         │ │
│  │ - 表: logs (主日志表)                                       │ │
│  │ - 索引: tokenbf_v1 全文索引, bloom_filter 标签索引          │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

### 2. 组件配置变更

#### 2.1 iLogtail 配置 (04-ilogtail.yaml)

**变更前**: `flusher_http` 直连 ClickHouse
**变更后**: `flusher_otlp` 发送到 OTEL Collector

```yaml
# 关键配置变更
flushers:
  - Type: flusher_otlp
    Endpoint: "otel-collector.edge-logs.svc.cluster.local:4317"
    Protocol: grpc
    TLS:
      Insecure: true
    Logs:
      ResourceAttributes:
        - From: "__host_name__"
          Key: "host.name"
        - From: "__host_ip__"
          Key: "host.ip"
```

#### 2.2 OTEL Collector 配置 (05-otel-collector.yaml)

**变更前**: 写入独立的 `otel_logs` 表
**变更后**: 写入现有的 `logs` 表，进行字段映射

```yaml
exporters:
  clickhouse:
    endpoint: tcp://clickhouse:9000
    database: edge_logs
    logs_table_name: logs  # 改为现有表
```

#### 2.3 ClickHouse 表结构

**保持现有 logs 表结构不变**，OTEL Collector 负责字段映射：

| OTEL 字段 | ClickHouse 字段 |
|-----------|-----------------|
| `Timestamp` | `timestamp` |
| `Body` | `content` |
| `SeverityText` | `severity` |
| `attributes["k8s.namespace.name"]` | `k8s_namespace_name` |
| `attributes["k8s.pod.name"]` | `k8s_pod_name` |
| `attributes["k8s.container.name"]` | `container_name` |
| `attributes["host.ip"]` | `host_ip` |
| `attributes["host.name"]` | `host_name` |

### 3. 参考实现 (APO 项目)

基于对 APO 项目的分析，其核心架构：

```sql
-- APO 的 ilogtail_logs 表结构
CREATE TABLE ilogtail_logs (
    timestamp          DateTime64(9),
    content            String,
    source             String,
    container_id       String,
    pid                String,
    container_name     LowCardinality(String),
    host_ip            LowCardinality(String),
    host_name          LowCardinality(String),
    k8s_namespace_name LowCardinality(String),
    k8s_pod_name       LowCardinality(String),
    INDEX idx_content content TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (host_ip, timestamp);
```

我们的 `logs` 表结构与 APO 兼容，只需确保 OTEL Collector 正确映射字段。

## 后果

### 优点

1. **标准化**: 遵循 OpenTelemetry 标准，与行业最佳实践一致
2. **解耦**: 日志采集器与存储系统解耦，易于替换组件
3. **可扩展**: OTEL Collector 支持多种 receivers/exporters，方便扩展
4. **数据处理能力**: 支持数据过滤、转换、聚合、采样
5. **可靠性**: 内置重试、批处理、背压控制
6. **可观测性**: OTEL Collector 自身提供丰富的监控指标

### 缺点

1. **架构复杂度增加**: 多一层中间件需要额外运维
2. **资源消耗增加**: OTEL Collector 需要额外的 CPU/内存
3. **延迟微增**: 数据经过中间层会有额外延迟（通常 <100ms）

### 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| OTEL Collector 成为单点故障 | 后续可扩展为多副本 + 负载均衡 |
| 配置复杂度高 | 提供标准化配置模板 |
| 字段映射错误 | 完整的端到端测试验证 |

## 实施计划

### 阶段 1: 配置修改
- [ ] 修改 iLogtail 配置为 OTLP gRPC 输出
- [ ] 调整 OTEL Collector 配置写入 logs 表
- [ ] 更新 ClickHouse 表结构（如需要）

### 阶段 2: 部署验证
- [ ] 在测试环境部署新配置
- [ ] 验证数据流完整性
- [ ] 检查字段映射正确性

### 阶段 3: 生产发布
- [ ] 滚动更新生产环境
- [ ] 监控数据采集指标
- [ ] 验证 API 查询正常

## 参考资料

- [APO 项目](https://github.com/CloudDetail/apo) - 参考架构来源
  - /Users/neov/src/github.com/CloudDetail/apo
- [iLogtail 文档](https://github.com/alibaba/ilogtail) - 日志采集器
  - /Users/neov/src/github.com/alibaba/ilogtail
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) - 官方文档
  - /Users/neov/src/github.com/open-telemetry/opentelemetry-collector
- [ClickHouse Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/clickhouseexporter) - OTEL ClickHouse 导出器
  - /Users/neov/src/github.com/open-telemetry/opentelemetry-collector-contrib

## 变更历史

| 日期 | 版本 | 变更内容 | 作者 |
|------|------|----------|------|
| 2026-01-12 | 1.0 | 初始版本 | Neov |
