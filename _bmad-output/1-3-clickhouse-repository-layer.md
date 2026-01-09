# Story 1.3: clickhouse-repository-layer

Status: done

<!-- 注意: 验证是可选的。在dev-story之前运行validate-create-story进行质量检查。 -->

## 用户故事

作为一名开发人员，
我希望实现一个 ClickHouse 仓储层，
以便我能够对日志数据执行 CRUD 操作，具有适当的错误处理和连接管理。

## 验收标准

**给定** ClickHouse 模式已设置
**当** 我实现仓储层
**那么** 我能够建立与 ClickHouse 的连接并具有适当的配置
**并且** 我能够将日志记录插入日志表
**并且** 我能够按数据集、时间范围和基本过滤器查询日志
**并且** 连接池和错误处理已实现
**并且** 所有数据库操作使用 klog/v2 进行结构化日志记录

## 任务/子任务

- [ ] 实现 ClickHouse 连接管理 (AC: 1)
  - [ ] 配置 ClickHouse 客户端连接池
  - [ ] 实现连接配置和参数验证
  - [ ] 添加连接健康检查和重试机制
  - [ ] 集成 klog/v2 结构化日志记录
- [ ] 实现日志记录插入操作 (AC: 2)
  - [ ] 创建 InsertLog 方法以支持 Story 1-2 模式
  - [ ] 实现批量插入优化以提高 iLogtail 性能
  - [ ] 添加数据验证和清理
  - [ ] 配置事务处理和错误恢复
- [ ] 实现核心日志查询操作 (AC: 3)
  - [ ] 实现按数据集和时间范围的 QueryLogs 方法
  - [ ] 添加基本过滤器（namespace、pod_name、severity）
  - [ ] 实现分页和结果限制
  - [ ] 优化使用 ClickHouse 索引和分区
- [ ] 实现高级查询构建器 (AC: 3)
  - [ ] 创建动态 WHERE 条件构建器
  - [ ] 实现全文搜索支持（tokenbf_v1 索引）
  - [ ] 添加标签过滤支持（tags['cluster'] 等）
  - [ ] 集成查询性能监控
- [ ] 实现连接池和资源管理 (AC: 4)
  - [ ] 配置 ClickHouse 连接池设置
  - [ ] 实现优雅关闭和资源清理
  - [ ] 添加连接泄漏检测
  - [ ] 实现连接重试和故障转移逻辑
- [ ] 实现全面的错误处理 (AC: 5)
  - [ ] 创建仓储特定的错误类型
  - [ ] 实现 ClickHouse 特定错误映射
  - [ ] 添加操作超时和取消支持
  - [ ] 集成结构化日志记录以便调试

## 开发说明

### 架构合规要求

**关键:** 此仓储层实现了与 Story 1-2 中创建的 ClickHouse 模式的直接集成。必须严格遵循 architecture.md 中指定的 APO 平台模式。

**关键技术要求:**
- **连接库:** clickhouse-go/v2 用于原生 ClickHouse 协议
- **连接池:** 配置最佳性能和资源利用率
- **日志记录:** klog/v2 用于与 edge-apiserver 一致的结构化日志
- **错误处理:** 全面的错误包装和上下文
- **性能:** 查询优化利用 Story 1-2 索引和分区

### ClickHouse 集成模式（基于 APO 平台）

**关键:** 使用确切的 Story 1-2 模式字段映射，针对边缘计算工作负载优化:

```go
// LogEntry 表示 ClickHouse 中的日志条目（映射到 Story 1-2 模式）
type LogEntry struct {
    // 时间和数据隔离
    Timestamp    time.Time         `ch:"timestamp"`
    Dataset      string            `ch:"dataset"`

    // 日志内容
    Content      string            `ch:"content"`
    Severity     string            `ch:"severity"`

    // 容器信息
    ContainerID  string            `ch:"container_id"`
    ContainerName string           `ch:"container_name"`
    PID          string            `ch:"pid"`

    // 主机信息
    HostIP       string            `ch:"host_ip"`
    HostName     string            `ch:"host_name"`

    // K8s 元数据
    K8sNamespace string            `ch:"k8s_namespace_name"`
    K8sPodName   string            `ch:"k8s_pod_name"`
    K8sPodUID    string            `ch:"k8s_pod_uid"`
    K8sNodeName  string            `ch:"k8s_node_name"`

    // 分析维度的标签
    Tags         map[string]string `ch:"tags"`
}
```

### 连接管理模式

**基于 clickhouse-go/v2 最佳实践:**

```go
// 连接配置对齐架构要求
type Config struct {
    // 连接设置
    Host         string   `yaml:"host"`
    Port         int      `yaml:"port"`
    Database     string   `yaml:"database"`
    Username     string   `yaml:"username"`
    Password     string   `yaml:"password"`

    // 连接池设置（边缘优化）
    MaxOpenConns    int           `yaml:"max_open_conns"`
    MaxIdleConns    int           `yaml:"max_idle_conns"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
    ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`

    // 查询设置
    QueryTimeout    time.Duration `yaml:"query_timeout"`
    ExecTimeout     time.Duration `yaml:"exec_timeout"`
}
```

### APO 查询优化模式

**利用 Story 1-2 性能索引:**

| 查询类型 | 优化策略 | 索引利用 |
|-----------|-----------|-----------|
| **按数据集查询** | WHERE dataset = ? 首先 | 分区键利用 |
| **时间范围查询** | ORDER BY 优化 (dataset, host_ip, timestamp) | 主键范围扫描 |
| **内容搜索** | 使用 tokenbf_v1 索引 | idx_content 全文搜索 |
| **标签过滤** | tags['cluster'] 查询 | idx_tags 布隆过滤器 |
| **命名空间/Pod 过滤** | LowCardinality 列优化 | 高效的基数查询 |

### 查询构建器实现模式

**动态查询构建遵循 ClickHouse 最佳实践:**

```go
// 查询构建器用于类型安全的 ClickHouse 查询
type QueryBuilder struct {
    baseQuery  strings.Builder
    conditions []string
    args       []interface{}
    orderBy    string
    limit      *int
    offset     *int
}

// 示例查询模式（利用 Story 1-2 优化）
func (qb *QueryBuilder) BuildLogQuery(req *LogQueryRequest) (string, []interface{}) {
    // 1. 基础查询，利用分区修剪
    qb.baseQuery.WriteString(`
        SELECT timestamp, dataset, content, severity,
               container_name, host_name,
               k8s_namespace_name, k8s_pod_name, k8s_node_name,
               tags
        FROM logs
    `)

    // 2. 数据集过滤（必需，用于分区修剪）
    qb.AddCondition("dataset = ?", req.Dataset)

    // 3. 时间范围（利用 ORDER BY 优化）
    if req.StartTime != nil {
        qb.AddCondition("timestamp >= ?", req.StartTime)
    }
    if req.EndTime != nil {
        qb.AddCondition("timestamp <= ?", req.EndTime)
    }

    // 4. K8s 元数据过滤（LowCardinality 优化）
    if req.Namespace != "" {
        qb.AddCondition("k8s_namespace_name = ?", req.Namespace)
    }

    // 5. 内容搜索（tokenbf_v1 索引）
    if req.Filter != "" {
        qb.AddCondition("hasToken(content, ?)", req.Filter)
    }

    // 6. 按 ORDER BY 键排序以获得最佳性能
    qb.SetOrderBy("dataset, host_ip, timestamp DESC")

    return qb.Build()
}
```

### 错误处理模式

**ClickHouse 特定错误映射和上下文:**

```go
// 仓储错误类型用于适当的错误分类
type RepositoryError struct {
    Op      string // 操作（QueryLogs、InsertLog）
    Table   string // 表名
    Err     error  // 原始错误
    Context map[string]interface{} // 额外上下文
}

// 常见 ClickHouse 错误到仓储错误的映射
func MapClickHouseError(err error, op string) *RepositoryError {
    if err == nil {
        return nil
    }

    // 检测常见的 ClickHouse 错误模式
    errMsg := err.Error()
    switch {
    case strings.Contains(errMsg, "connection refused"):
        return &RepositoryError{
            Op: op, Err: err,
            Context: map[string]interface{}{"type": "connection_error"},
        }
    case strings.Contains(errMsg, "timeout"):
        return &RepositoryError{
            Op: op, Err: err,
            Context: map[string]interface{}{"type": "timeout_error"},
        }
    default:
        return &RepositoryError{Op: op, Err: err}
    }
}
```

### klog/v2 结构化日志模式

**与 edge-apiserver 一致的日志记录:**

```go
// 仓储操作的结构化日志记录
func (r *ClickHouseRepository) QueryLogs(ctx context.Context, req *LogQueryRequest) ([]LogEntry, int, error) {
    startTime := time.Now()

    klog.InfoS("开始日志查询",
        "dataset", req.Dataset,
        "start_time", req.StartTime,
        "end_time", req.EndTime,
        "namespace", req.Namespace)

    // 执行查询...

    duration := time.Since(startTime)
    if err != nil {
        klog.ErrorS(err, "日志查询失败",
            "dataset", req.Dataset,
            "duration_ms", duration.Milliseconds(),
            "error_type", reflect.TypeOf(err).String())
        return nil, 0, err
    }

    klog.InfoS("日志查询完成",
        "dataset", req.Dataset,
        "returned_rows", len(results),
        "total_rows", total,
        "duration_ms", duration.Milliseconds())

    return results, total, nil
}
```

### 性能要求和监控

**查询性能目标（NFR1）:**
- 标准时间范围查询: < 2秒
- 内容搜索查询: < 3秒（利用 tokenbf_v1）
- 批量插入: > 10,000 记录/秒（iLogtail 优化）
- 连接池利用率: > 80% 效率

**监控集成:**
```go
// 查询指标跟踪
func (r *ClickHouseRepository) recordQueryMetrics(op string, duration time.Duration, err error) {
    // 记录到查询统计表（Story 1-2 中创建）
    stats := &QueryStats{
        QueryID:         generateQueryID(),
        Dataset:         dataset,
        QueryType:       op,
        ExecutionTimeMs: uint32(duration.Milliseconds()),
        CreatedAt:       time.Now(),
    }

    // 异步插入以避免影响主查询路径
    go r.insertQueryStats(context.Background(), stats)
}
```

### 测试策略

**全面测试覆盖符合质量标准:**

1. **单元测试:**
   - 连接管理和配置验证
   - 查询构建器逻辑和 SQL 生成
   - 错误处理和错误映射
   - 数据转换和验证

2. **集成测试（使用 testcontainers）:**
   - 真实 ClickHouse 实例测试
   - Story 1-2 模式兼容性验证
   - 性能基准测试（查询时间 < 2秒）
   - 并发连接和连接池测试

3. **边缘情况测试:**
   - 网络中断和重连
   - 大型结果集处理
   - 内存压力下的连接池行为
   - 恶意查询保护

### 项目结构遵循

**文件组织与架构对齐:**
```
pkg/repository/clickhouse/
├── repository.go          # 主仓储实现
├── connection.go          # 连接管理和池
├── query_builder.go       # 动态查询构建
├── error.go              # 错误处理和映射
├── metrics.go            # 查询指标记录
├── repository_test.go    # 单元测试
├── integration_test.go   # 集成测试（testcontainers）
└── benchmark_test.go     # 性能基准测试
```

**关键集成点:**
- 必须使用 pkg/model/clickhouse/log.go 中的现有 LogEntry 结构
- 必须使用 pkg/model/request/log.go 中的现有 LogQueryRequest
- 必须与 pkg/config/config.go 中的配置系统集成
- 必须支持 pkg/middleware/ 中的日志和指标中间件

### 依赖项和版本要求

**关键依赖项（与 edge-apiserver 对齐）:**
```go
// go.mod 依赖项
require (
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0
    k8s.io/klog/v2 v2.100.1
    github.com/stretchr/testify v1.8.4
    github.com/testcontainers/testcontainers-go v0.26.0
)
```

### iLogtail 集成就绪

**为 iLogtail 直接写入准备:**
- 插入方法必须处理 iLogtail 批量操作
- 字段映射必须支持 architecture.md 中指定的所有 iLogtail 字段
- 性能必须支持高吞吐量边缘日志摄取
- 错误处理必须是宽松的以避免日志丢失

### 安全考虑

**数据访问安全:**
- 数据集隔离在存储库级别强制执行
- SQL 注入保护通过参数化查询
- 连接凭据通过配置管理安全处理
- 查询超时防止资源耗尽攻击

### 参考文档

- [来源: _bmad-output/architecture.md#仓储层] - 完整的仓储层架构规范
- [来源: _bmad-output/architecture.md#ClickHouse Schema 设计] - Schema 和性能要求
- [来源: _bmad-output/epics.md#Story 1.3] - 用户故事和验收标准
- [来源: _bmad-output/1-2-clickhouse-database-schema-setup.md] - 数据库模式依赖项
- [来源: sqlscripts/clickhouse/01_tables.sql] - 确切的表结构和索引
- [来源: pkg/repository/clickhouse/repository.go] - 现有接口和结构

## 开发智能体记录

### 使用的智能体模型

claude-sonnet-4-20250514

### 调试日志参考

基于 Story 1-2 完整数据库模式构建 - 参考先前故事的 ClickHouse 实现细节以获得模式兼容性。

### 完成说明列表

### 文件列表