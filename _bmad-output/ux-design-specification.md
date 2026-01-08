---
stepsCompleted: [1, 2, 3, 4, 5, 6]
inputDocuments: ["_bmad-output/architecture.md"]
workflowType: 'ux-design'
lastStep: 6
project_name: 'edge-logs'
user_name: 'Neov'
date: '2026-01-08'
---

# UX Design Specification - edge-logs

**Author:** Neov
**Date:** 2026-01-08

---

## Executive Summary

### Project Vision

构建一个基于 ClickHouse 的云原生日志平台，专注于边缘计算场景。通过 K8s API Aggregation 模式提供统一的日志查询 API，支持多 dataset 数据隔离和多集群/节点维度的日志分析。

### Target Users

**主要用户群体：**

- **云平台运维人员** - 需要跨多个边缘集群查看和排查问题
- **SRE/站点可靠性工程师** - 需要 7x24 监控和快速故障定位
- **Kubernetes 集群管理员** - 管理边缘节点和日志采集

**用户特征：**
- 熟悉 Kubernetes 和云原生技术栈
- 有日志查询经验（如 ELK、Loki）
- 期望高性能查询和直观的过滤能力
- 工作环境：桌面浏览器、监控大屏

### Key Design Challenges

1. **数据量与性能平衡**
   - 日志数据量庞大，需要高效的分页、加载、渲染策略
   - 大量数据下的查询响应时间优化
   - 浏览器性能考虑（虚拟滚动、懒加载）

2. **多 Dataset 切换体验**
   - 如何让用户方便地在不同数据集间切换
   - 切换时保持查询上下文或状态
   - 清晰的数据集标识和层级展示

3. **复杂查询简化**
   - 如何简化日志过滤语法
   - 提供常用查询条件的快捷方式
   - 高级查询与简单查询的平衡

### Design Opportunities

1. **统一聚合视图**
   - 一次查询，跨多个集群/节点聚合显示
   - 按时间线自动归并不同数据源的日志
   - 智能去重和时序对齐

2. **智能过滤体验**
   - 常用查询条件的快捷方式（severity、namespace、pod）
   - 搜索历史和保存的查询
   - 自动补全和提示

---

## Core User Experience

### Defining Experience

edge-logs 的核心体验是**快速、精准地查询和过滤日志**。

用户的核心循环：
1. 选择 dataset（数据集）
2. 设置查询条件（时间、过滤条件）
3. 查看结果列表
4. 点击查看日志详情
5. 返回并调整查询

### Platform Strategy

**平台决策：**
- **主要平台**：Web 应用（桌面浏览器）
- **响应式设计**：支持 1920px+ 大屏显示
- **交互方式**：鼠标/键盘为主，支持快捷键
- **无需离线**：云端查询，始终在线

**浏览器要求：**
- Chrome 90+、Firefox 88+、Safari 14+、Edge 90+
- 支持 WebSocket（可选，用于实时状态更新）

### Effortless Interactions

**应该完全自然的交互：**

1. **查询历史**
   - 自动保存最近 10 条查询
   - 一键重放历史查询

2. **快捷过滤**
   - severity 标签式选择（Error/Warning/Info/Debug）
   - namespace/pod 自动补全
   - 时间范围快捷选项（15分钟、1小时、今天、昨天）

3. **结果定位**
   - 自动高亮搜索关键词
   - 滚动时自动定位到匹配项

4. **智能建议**
   - 根据输入提示相关字段名
   - 显示查询模板

### Critical Success Moments

**关键成功时刻：**

1. **首查即中** - 新用户第一次查询就能找到目标日志
2. **速度感知** - 查询结果在 2 秒内返回，明显快于竞品
3. **精准过滤** - 多条件组合查询一次命中
4. **无缝切换** - 在不同 dataset 间切换，查询条件保持

**成败关键：**
- 时间范围过滤必须精确（毫秒级）
- 多 dataset 切换不能混淆数据
- 大数据量下页面不卡顿

### Experience Principles

**指导原则：**

1. **速度至上** - 查询响应时间 < 2秒，否则显示加载状态
2. **渐进式复杂** - 默认简单查询，高级选项按需展开
3. **上下文保持** - 切换 dataset/页面时保留查询状态
4. **视觉清晰** - 日志层级、时间线、高亮一目了然

---

## Desired Emotional Response

### Primary Emotional Goals

edge-logs 的核心情感目标是让用户感受到**掌控感**和**效率感**。

- **掌控感** - 清晰知道数据来自哪里，查询结果准确可靠
- **效率感** - 快速找到目标日志，问题解决时间大幅缩短

### Emotional Journey Mapping

| 阶段 | 用户情感 | 设计支持 |
|------|----------|----------|
| 首次使用 | "界面简洁，看起来很专业" | 清晰的布局，明确的操作入口 |
| 选择数据集 | "知道自己在查什么" | dataset 标识清晰，层级明确 |
| 执行查询 | 期待但不确定 | 查询中状态提示，预计时间 |
| 查看结果 | "找到了！就是这条" | 关键词高亮，时间线清晰 |
| 完成任务 | 成就感、轻松感 | 问题解决，可以继续工作 |
| 再次使用 | 习惯、依赖 | 查询历史，快捷操作 |

### Micro-Emotions

**关键的微观情感状态：**

1. **自信 vs 困惑**
   - ✅ 清晰的界面布局和标签
   - ✅ 操作反馈明确
   - ❌ 隐藏的选项或模糊的标签

2. **信任 vs 怀疑**
   - ✅ 显示结果数量和查询耗时
   - ✅ dataset 标识始终可见
   - ❌ 结果来源不透明

3. **成就感 vs 挫败感**
   - ✅ 搜索关键词自动高亮
   - ✅ 滚动时定位到匹配项
   - ❌ 需要手动翻页查找

4. **掌控感 vs 无力感**
   - ✅ 丰富的过滤选项
   - ✅ 查询历史重放
   - ❌ 只能用简单搜索

### Design Implications

**情感-设计连接：**

| 情感目标 | UX 设计方法 |
|----------|-------------|
| 掌控感 | • dataset 切换器始终可见<br>• 查询条件实时显示<br>• 结果数量和时间明确 |
| 效率感 | • 快捷键支持<br>• 查询历史一键重放<br>• 自动补全和智能建议 |
| 信任感 | • 查询时间显示<br>• 数据来源标识<br>• 准确的结果计数 |
| 专注感 | • 简洁的界面布局<br>• 无干扰的设计<br>• 清晰的视觉层级 |
| 安心感 | • 明确的数据隔离<br>• 切换 dataset 时状态保持 |

### Emotional Design Principles

**情感设计指导原则：**

1. **速度即信任** - 快速响应建立用户信任
2. **透明即掌控** - 数据来源和查询状态始终可见
3. **简约即专业** - 技术用户也需要简洁
4. **反馈即安心** - 每个操作都有明确反馈

---

## UX Pattern Analysis & Inspiration

### Inspiring Products Analysis

**Kibana (Elasticsearch)**
- **核心优势**：强大的发现栏 + 可视化配置
- **导航体验**：左侧索引模式 + 时间范围选择器
- **创新交互**：查询自动保存、字段自动探索
- **视觉设计**：深色主题、高对比度日志显示

**Grafana Loki**
- **核心优势**：LogQL 查询 + Grafana 统一体验
- **导航体验**：与监控指标一致的界面
- **创新交互**：标签过滤的即时反馈
- **视觉设计**：简洁的查询编辑器

**CloudWatch Logs Insights**
- **核心优势**：类 SQL 查询语法
- **导航体验**：多日志组切换
- **创新交互**：查询结果导出、分析模板

### Transferable UX Patterns

| 模式 | 来源 | 应用到 edge-logs |
|------|------|------------------|
| 时间范围快捷选择 | Kibana/Loki | 15m/1h/今日/昨日快捷按钮 |
| 查询历史自动保存 | Kibana | 最近 10 条查询，一键重放 |
| 字段值自动补全 | Kibana | namespace/pod 自动补全 |
| Severity 标签过滤 | Loki | Error/Warning/Info 标签式选择 |
| 结果高亮定位 | Kibana | 关键词自动高亮 |
| 深色主题支持 | Kibana/Grafana | 专业工具标配 |

### Anti-Patterns to Avoid

| 反模式 | 问题 |
|--------|------|
| 复杂的查询 DSL | 学习成本高，普通用户不会用 |
| 查询结果无限制返回 | 页面卡顿，浏览器崩溃 |
| 过度嵌套的配置项 | 界面混乱，找不到功能 |
| 无加载状态提示 | 用户不确定是否在查询 |
| 时间选择器不直观 | 难以快速选择范围 |

### Design Inspiration Strategy

**采用（Adopt）：**
- 时间范围快捷按钮 - 行业标准，用户熟悉
- Severity 标签式过滤 - 直观高效
- 查询历史保存 - 减少重复输入
- 深色主题 - 技术用户期望

**适配（Adapt）：**
- 简化查询语法 - 使用简单的键值对过滤，而非复杂 DSL
- 虚拟滚动 - 处理大数据量，避免一次性加载

**避免（Avoid）：**
- 复杂的查询编辑器 - 与我们的"简洁"原则冲突
- 无限分页 - 改用虚拟滚动
- 过度的��置选项 - 保持界面简洁

---

## Design System Foundation

### Design System Choice

**Ant Design (React) v5**

选择 Ant Design 作为 edge-logs UI 的设计系统基础。

### Rationale for Selection

| 因素 | 说明 |
|------|------|
| **丰富的数据组件** | Table、Form、Select、DatePicker 等适合日志查询场景 |
| **深色主题内置** | 专业工具标配，技术用户期望 |
| **优秀的文档** | 降低学习成本，快速开发 |
| **TypeScript 支持** | 与现代化技术栈一致，类型安全 |
| **可定制性** | CSS Variables 主题定制，灵活调整 |
| **性能优化** | 虚拟滚动等性能优化组件内置 |

### Implementation Approach

**技术栈：**
- React 18+ / Vue 3+（前端框架）
- Ant Design v5（UI 组件库）
- TypeScript（类型安全）
- CSS-in-JS 或 Tailwind CSS（样式）

**核心组件使用：**

| Ant Design 组件 | 用途 |
|-----------------|------|
| Table | 日志列表展示（虚拟滚动） |
| Form | 查询条件表单 |
| Select / AutoComplete | dataset、namespace 选择 |
| RangePicker | 时间范围选择 |
| Tag | severity、过滤条件标签 |
| Input.Search | 日志内容搜索 |
| Layout | 页面布局（Header + Sidebar + Content） |
| Typography | 字体和排版 |

### Customization Strategy

**主题定制：**

```javascript
// theme.config.js
export default {
  token: {
    // 主色：科技蓝
    colorPrimary: '#1677ff',
    colorBgContainer: '#141414',
    colorText: '#ffffff',

    // 深色主题
    algorithm: true, // 启用深色算法
  },
  components: {
    Table: {
      // 表格定制
      headerBg: '#1f1f1f',
      rowHoverBg: '#262626',
    },
    Input: {
      // 输入框定制
      colorBgContainer: '#1f1f1f',
    },
  },
};
```

**自定义组件：**

1. **LogViewer** - 日志内容查看器
   - 语法高亮（severity 颜色）
   - 关键词高亮
   - 时间戳显示

2. **DatasetSelector** - 数据集选择器
   - 树形或标签页结构
   - 多层级支持（cluster → dataset）

3. **QueryBuilder** - 查询构建器
   - 可视化过滤条件
   - 查询历史下拉

**性能优化：**
- 使用虚拟滚动（rc-virtual-list）处理大数据量
- 懒加载日志详情
- 防抖搜索输入
