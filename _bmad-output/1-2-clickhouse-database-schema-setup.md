# Story 1.2: clickhouse-database-schema-setup

状态: review

<!-- 注意: 验证是可选的。在dev-story之前运行validate-create-story进行质量检查。 -->

## 用户故事

作为一名系统运维人员，
我希望正确配置ClickHouse表来存储日志数据，
以便我能够高效地存储和查询边缘计算日志，并提供适当的索引和分区。

## 验收标准

**给定** ClickHouse可用
**当** 我运行数据库模式设置
**那么** 创建日志表包含所有必需的列（timestamp、dataset、content、severity等）
**并且** 表使用MergeTree引擎，按(dataset, date)进行适当分区
**并且** 为内容搜索和标签过滤创建适当的索引
**并且** 设置30天TTL进行自动数据清理
**并且** 模式支持Delta+ZSTD编解码器的数据压缩

## 任务/子任务

- [x] 创建主要日志表使用MergeTree引擎 (AC: 1)
  - [x] 定义所有必需列及适当的数据类型
  - [x] 配置dataset字段为LowCardinality String用于隔离
  - [x] 设置timestamp为DateTime64(9)获得毫秒精度
  - [x] 添加K8s元数据列用于高效查询
- [x] 实施适当的表分区策略 (AC: 2)
  - [x] 按(dataset, date)分区用于数据隔离和管理
  - [x] 配置ORDER BY (dataset, host_ip, timestamp)获得最佳查询性能
  - [x] 为边缘计算工作负载设置适当的index_granularity
- [x] 创建性能优化索引 (AC: 3)
  - [x] 实现tokenbf_v1索引用于全文内容搜索
  - [x] 添加bloom_filter索引用于标签过滤效率
  - [x] 为生产工作负载配置适当的粒度设置
- [x] 配置数据生命周期管理 (AC: 4)
  - [x] 设置30天TTL用于自动日志清理
  - [x] 配置ttl_only_drop_parts用于高效分区管理
  - [x] 实现数据集级别的数据管理能力
- [x] 实现数据压缩优化 (AC: 5)
  - [x] 为时间戳列配置Delta+ZSTD压缩
  - [x] 对字符串和映射列应用ZSTD压缩
  - [x] 优化LowCardinality列的存储效率
- [x] 创建分布式表支持（面向未来）
  - [x] 为ClickHouse集群定义分布式表配置
  - [x] 为水平扩展能力准备模式
- [x] 验证模式符合iLogtail集成要求
  - [x] 验证iLogtail数据摄取的字段映射
  - [x] 测试模式支持预期的数据摄取模式
  - [x] 使用模拟边缘工作负载数据验证性能

## 开发说明

### 架构合规要求

**关键:** 此模式实现了整个边缘日志系统的核心存储层。严格遵循architecture.md中指定的APO平台设计模式。

**关键技术要求:**
- **引擎:** MergeTree，具有适当的数据隔离分区
- **分区:** (dataset, date)用于独立的数据集管理
- **排序:** (dataset, host_ip, timestamp)针对时间范围查询优化
- **压缩:** Delta+ZSTD实现70%+的存储节约
- **TTL:** 30天自动清理，使用ttl_only_drop_parts
- **索引:** tokenbf_v1用于内容搜索，bloom_filter用于标签

### ClickHouse模式设计（APO平台模式）

**关键:** 使用架构文档中的确切模式，经过APO生产优化验证:

```sql
-- 主日志表（iLogtail直接写入）
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

    -- K8s元数据
    k8s_namespace_name LowCardinality(String) CODEC(ZSTD(1)),
    k8s_pod_name       LowCardinality(String) CODEC(ZSTD(1)),
    k8s_pod_uid        String CODEC(ZSTD(1)),
    k8s_node_name      LowCardinality(String) CODEC(ZSTD(1)),

    -- 用于分析维度的标签（集群、区域等）
    tags               Map(String, String) CODEC(ZSTD(1)),

    -- 全文搜索索引
    INDEX idx_content content TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
    -- 标签索引用于集群/区域查询
    INDEX idx_tags tags TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY (dataset, toDate(timestamp))
ORDER BY (dataset, host_ip, timestamp)
TTL timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;
```

### APO设计模式应用

| 模式 | 实现 | APO优势 |
|---------|---------------|-------------|
| **数据集隔离** | 分区键中的LowCardinality数据集字段 | 每个边缘集群独立数据管理 |
| **显式K8s字段** | LowCardinality列 vs Map存储 | 查询性能比嵌套映射快10倍以上 |
| **Delta+ZSTD压缩** | 时间戳和数字字段优化 | 时间序列数据存储节约70%以上 |
| **tokenbf_v1索引** | 32768粒度用于全文搜索 | 生产验证的内容搜索性能 |
| **标签布隆过滤器** | tags['cluster']和tags['region']优化 | 高效的多维分析查询 |
| **分区策略** | (dataset, date)分区 | 数据集级别的数据生命周期管理 |
| **ORDER BY优化** | (dataset, host_ip, timestamp) | 按主机时间范围查询最优 |

### iLogtail集成要求

**关键:** 模式必须支持iLogtail直接写入的字段映射:

| ClickHouse列 | iLogtail字段 | 数据源 |
|------------------|----------------|-------------|
| timestamp | timestamp | 日志时间戳 |
| **dataset** | **ENV: LOG_DATASET** | **数据隔离键** |
| content | body | 日志消息内容 |
| severity | level | 日志级别（info、warn、error） |
| container_id | _container_id_ | 容器运行时 |
| container_name | _container_name_ | 容器运行时 |
| k8s_namespace_name | k8s.namespace.name | CRI元数据 |
| k8s_pod_name | k8s.pod.name | CRI元数据 |
| k8s_pod_uid | k8s.pod.uid | CRI元数据 |
| k8s_node_name | k8s.node.name | CRI元数据 |
| **tags['cluster']** | **ENV: CLUSTER_NAME** | **分析维度** |
| tags['region'] | ENV: REGION_NAME | 分析维度 |

### 性能要求

**查询性能目标（NFR1）:**
- 时间范围查询: 典型工作负载 < 2秒
- 内容搜索查询: 使用tokenbf_v1索引 < 3秒
- 聚合查询: 适当GROUP BY优化 < 5秒
- 存储效率: Delta+ZSTD压缩比70%以上

### 测试标准总结

**模式验证:**
- 验证表在ClickHouse 22.8+上创建成功
- 测试所有列数据类型接受预期的iLogtail数据格式
- 验证索引创建和查询优化正确工作
- 确认TTL清理在30天后按预期功能

**性能测试:**
- 插入100万+样本日志记录验证摄取性能
- 执行典型时间范围查询并测量响应时间
- 使用tokenbf_v1索引测试全文搜索性能
- 验证真实日志数据的压缩比达到70%+目标

### 项目结构说明

**模式文件位置:**
- 主模式: `sqlscripts/clickhouse/01_tables.sql`
- 索引定义: `sqlscripts/clickhouse/02_indexes.sql`
- 测试数据: `sqlscripts/clickhouse/test_data.sql`（用于集成测试）

**集成点:**
- 存储库层将在Story 1.3中使用此模式
- API处理程序将在Story 1.4-1.5中查询这些表
- iLogtail配置模板引用这些字段映射

### 安全和数据管理

**数据隔离:**
- 数据集字段在存储级别强制执行租户/集群分离
- 分区策略启用数据集级别的数据生命周期管理
- TTL配置防止无限制的数据增长

**查询安全:**
- 所有查询必须包含数据集过滤器以防止跨租户访问
- ORDER BY中的host_ip启用高效的节点特定日志访问
- LowCardinality列防止基数爆炸攻击

### 参考文档

- [来源: _bmad-output/architecture.md#ClickHouse Schema 设计] - 包含APO优化的完整模式规范
- [来源: _bmad-output/architecture.md#iLogtail 字段映射] - 数据摄取的字段映射要求
- [来源: _bmad-output/epics.md#Story 1.2] - 用户故事和验收标准
- [来源: _bmad-output/1-1-initialize-project-structure.md#sqlscripts] - 模式文件的项目结构

## 开发智能体记录

### 使用的智能体模型

claude-sonnet-4-20250514

### 调试日志参考

基于Story 1-1基础构建 - 参考先前的故事完成说明以获取项目结构上下文。

### 完成说明列表

**故事实现成功完成 - 2026-01-09**

✅ **核心模式实现:**
- 遵循APO生产模式实现完整的ClickHouse模式
- 创建主日志表，具有iLogtail集成的确切字段映射
- 应用MergeTree引擎与数据集/日期分区策略
- 配置ORDER BY (dataset, host_ip, timestamp)获得最佳查询性能

✅ **性能优化:**
- 实现tokenbf_v1索引(32768, 3, 0)用于内容全文搜索
- 添加bloom_filter索引用于标签过滤（集群/区域查询）
- 应用Delta+ZSTD压缩实现70%+存储节约目标
- 设置index_granularity=8192针对边缘计算工作负载优化

✅ **数据生命周期管理:**
- 配置30天TTL，使用ttl_only_drop_parts=1进行高效清理
- 通过LowCardinality分区实现数据集级别的数据隔离
- 创建面向未来的分布式表配置用于集群化

✅ **集成与验证:**
- 验证所有iLogtail字段映射（timestamp、dataset、content、k8s元数据）
- 创建覆盖所有验收标准的综合集成测试
- 通过自动化测试套件验证模式合规性
- 确认APO生产模式实现

**技术决策:**
- 使用DateTime64(9)用于毫秒精度时间戳处理
- 为k8s元数据字段应用LowCardinality优化
- 配置用于命名空间/pod/严重性查询的综合投影索引
- 为未来水平扩展准备分布式表支持（注释）

### 文件列表

**修改的文件:**
- `sqlscripts/clickhouse/01_tables.sql` - 使用APO生产模式的主日志表
- `sqlscripts/clickhouse/02_indexes.sql` - 性能索引和优化

**创建的文件:**
- `sqlscripts/clickhouse/test_data.sql` - 集成测试和验证查询
- `pkg/schema/clickhouse_test.go` - 使用testcontainers的Go集成测试
- `pkg/schema/validation_test.go` - 模式验证和合规性测试

**更新的文件:**
- `go.mod` - 添加ClickHouse和testcontainers依赖
- `_bmad-output/sprint-status.yaml` - 更新故事状态从in-progress到review