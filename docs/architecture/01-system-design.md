# Token Bridge Intelligence - 架构设计

## 系统架构图

### 旧执行链（价格爬虫）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Token Bridge Crawler                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         cmd/crawler/main.go                          │   │
│  │                    (程序入口、配置加载、生命周期管理)                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      internal/scheduler.go                           │   │
│  │              (调度器：限流控制、降级策略、任务编排)                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│         ┌──────────────────────────┼──────────────────────────┐             │
│         ▼                          ▼                          ▼             │
│  ┌─────────────┐            ┌─────────────┐            ┌─────────────┐     │
│  │   adapters  │            │   storage   │            │  ai/mail    │     │
│  │  (厂商适配器) │            │  (数据存储)  │            │  (报告生成)  │     │
│  ├─────────────┤            ├─────────────┤            ├─────────────┤     │
│  │  google.go  │            │             │            │ summarizer  │     │
│  │  openai.go  │◄──────────►│  PostgreSQL │            │   .go       │     │
│  │ anthropic.go│            │             │            │  sender.go  │     │
│  └─────────────┘            └─────────────┘            └─────────────┘     │
│         │                          │                          │             │
│         ▼                          ▼                          ▼             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         internal/tbclient.go                         │   │
│  │                    (TB API 客户端：数据导入)                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         TB 主项目 API                                 │   │
│  │              POST /v1/admin/supplier_catalog_staging/import          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 新执行链（情报系统 + 决策层）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Token Bridge Intelligence                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       cmd/intelligence/main.go                       │   │
│  │                    (程序入口、配置加载、生命周期管理)                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│         ┌──────────────────────────┼──────────────────────────┐             │
│         ▼                          ▼                          ▼             │
│  ┌─────────────┐            ┌─────────────┐            ┌─────────────┐     │
│  │  scheduler  │            │  storage    │            │  reporter   │     │
│  │  (调度器)    │            │  (情报存储)  │            │  (报告生成)  │     │
│  └─────────────┘            └─────────────┘            └─────────────┘     │
│         │                          │                          │             │
│         ▼                          ▼                          ▼             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           collectors                               │   │
│  │  (多类型采集器：价格、API文档、用户痛点、工具生态)                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           marketing                                 │   │
│  │  (信号检测、动作生成、内部建议层)                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                             ai                                      │   │
│  │  (翻译服务、总结生成)                                                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 模块职责

### 1. 入口层

| 文件 | 职责 | 状态 |
|------|------|------|
| `cmd/crawler/main.go` | 旧价格爬虫入口 | 维护中 |
| `cmd/intelligence/main.go` | 新情报系统入口 | 主流程 |

**核心流程**:
1. 加载 `.env` 环境变量
2. 初始化存储、采集器、报告生成器、营销信号模型
3. 启动 Cron 定时任务或执行单次任务
4. 处理营销信号并持久化（仅在单次模式下）

### 2. 调度层

| 文件 | 功能 | 状态 |
|------|------|------|
| `internal/scheduler.go` | 旧调度器 | 维护中 |
| `internal/scheduler/intelligence.go` | 新情报调度器 | 主流程 |

**功能**:
- **任务调度**：按配置的 Cron 表达式触发采集任务
- **限流控制**：厂商间请求间隔控制（防封）
- **降级策略**：连续失败后自动调整采集频率
- **状态追踪**：记录每个采集器的运行状态

### 3. 采集器层 (internal/collectors/)

| 类型 | 采集器 | 功能 | 状态 |
|------|--------|------|------|
| 价格 | `price/google.go` | Google Gemini 价格采集 | ✅ 可用 |
| 价格 | `price/openai.go` | OpenAI 价格采集 | ✅ 可用 |
| 价格 | `price/anthropic.go` | Anthropic 价格采集 | ✅ 可用 |
| 价格 | `price/openrouter.go` | OpenRouter 市场情报 | ✅ 可用 |
| API文档 | `apidoc/openai.go` | OpenAI API文档监控 | ✅ 可用 |
| 用户痛点 | `userpain/hackernews.go` | HackerNews 监控 | ✅ 可用 |
| 用户痛点 | `userpain/reddit.go` | Reddit 监控 | ✅ 可用 |
| 用户痛点 | `userpain/configpain.go` | 配置痛点采集 | ✅ 可用 |
| 工具生态 | `tool/ecosystem.go` | 工具生态采集 | ✅ 可用 |
| 社区 | `community/discord.go` | Discord 消息采集 | ✅ **真实API** |
| 社区 | `community/linkedin.go` | LinkedIn 内容采集 | ⚠️ 模拟数据 |
| 社区 | `community/producthunt.go` | Product Hunt 产品采集 | ⚠️ 模拟数据 |

**采集器接口**:
```go
type Collector interface {
    Name() string                    // 采集器名称
    IntelType() core.IntelType       // 情报类型
    Source() string                  // 数据源
    Fetch(ctx context.Context) ([]core.IntelItem, error)  // 采集数据
    RateLimit() time.Duration        // 请求间隔
}
```

### 4. 存储层 (internal/storage/)

| 文件 | 功能 | 状态 |
|------|------|------|
| `storage.go` | 旧价格存储 | 维护中 |
| `intelligence.go` | 新情报存储 | 主流程 |
| `translated_storage.go` | 翻译存储包装 | 主流程 |

**功能**:
- **情报存储**：统一存储多类型情报
- **信号存储**：存储客户信号
- **动作存储**：存储营销动作
- **采集器运行日志**：记录每次采集的状态
- **告警历史**：存储告警记录
- **统计查询**：支持情报统计分析

### 5. 营销决策层 (internal/marketing/)

| 模块 | 功能 | 状态 |
|------|------|------|
| `types/` | 类型定义 | ✅ 已实现 |
| `detectors/` | 信号检测器 | ✅ 已实现 |
| `generators/` | 动作生成器 | ✅ 已实现 |
| `signal_model.go` | 信号模型 | ✅ 已实现 |

**功能**:
- **信号检测**：识别客户痛点和需求信号
- **动作生成**：基于信号生成内部建议
- **资格评估**：评估信号价值和客户阶段

**语义规范**:
- 所有动作为**内部建议层**
- `Channel: internal`
- `AutoExecute: false`
- `Status: draft`

### 6. 报告层 (internal/reporter/)

| 文件 | 功能 | 状态 |
|------|------|------|
| `daily.go` | 日报生成器 | ✅ 可用 |

**功能**:
- **日报生成**：自动生成每日情报摘要
- **邮件发送**：支持邮件通知

### 7. AI 服务层 (internal/ai/)

| 文件 | 功能 | 状态 |
|------|------|------|
| `translator.go` | 翻译服务 | ✅ 可用 |
| `translation_service.go` | 翻译服务管理 | ✅ 可用 |

**功能**:
- **文本翻译**：支持英文到中文的翻译
- **批量处理**：优化翻译请求

## 数据表结构

### 核心表

```sql
-- 情报项表
CREATE TABLE intelligence_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intel_type TEXT NOT NULL,
    source TEXT NOT NULL,
    source_id TEXT,
    title TEXT,
    content TEXT,
    url TEXT,
    metadata JSONB DEFAULT '{}',
    captured_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    status TEXT DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 客户信号表
CREATE TABLE customer_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intel_item_id UUID REFERENCES intelligence_items(id),
    signal_type TEXT NOT NULL,
    strength INT NOT NULL DEFAULT 1,
    content TEXT,
    platform TEXT,
    author TEXT,
    url TEXT,
    metadata JSONB DEFAULT '{}',
    status TEXT DEFAULT 'new',
    detected_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 营销动作表
CREATE TABLE marketing_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_type TEXT NOT NULL,
    channel TEXT NOT NULL,
    title TEXT,
    content TEXT,
    template_id TEXT,
    target_audience TEXT,
    priority INT NOT NULL DEFAULT 3,
    signal_ids JSONB DEFAULT '[]',
    auto_execute BOOLEAN DEFAULT FALSE,
    customer_stage TEXT,
    qualified_score NUMERIC(5,2),
    metadata JSONB DEFAULT '{}',
    status TEXT DEFAULT 'draft',
    scheduled_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 辅助表

```sql
-- 采集器运行记录
CREATE TABLE collector_runs (
    id UUID PRIMARY KEY,
    collector_name TEXT NOT NULL,
    intel_type TEXT NOT NULL,
    source TEXT NOT NULL,
    status TEXT NOT NULL,
    items_count INT DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_ms INT
);

-- 告警历史
CREATE TABLE alert_history (
    id UUID PRIMARY KEY,
    rule_id UUID,
    rule_name TEXT NOT NULL,
    intel_item_id UUID,
    intel_type TEXT NOT NULL,
    source TEXT NOT NULL,
    title TEXT,
    content TEXT,
    severity TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    notified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 数据流转

### 新执行链（情报系统 + 决策层）

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  外部数据源  │────►│  采集器     │────►│  存储层     │
│  (网站/API)  │     │ (多类型)    │     │ (统一存储)  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  报告生成   │◄────│  调度器     │◄────│  情报处理   │
│  (日报)     │     │ (定时任务)  │     │ (清洗/去重) │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  动作存储   │◄────│  动作生成   │◄────│  信号检测   │
│ (内部建议)  │     │ (内部建议)  │     │ (客户信号)  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  人工审核   │◄────│  翻译服务   │◄────│  AI 服务    │
│ (非自动)    │     │ (EN→CN)     │     │ (总结/翻译) │
└─────────────┘     └─────────────┘     └─────────────┘
```

## 配置体系

```
┌─────────────────────────────────────────┐
│           配置优先级（从高到低）          │
├─────────────────────────────────────────┤
│ 1. 环境变量 (CRAWLER_DATABASE_URL)      │
│ 2. .env 文件                            │
│ 3. config.yaml 文件                     │
│ 4. 代码默认值                            │
└─────────────────────────────────────────┘
```

## 错误处理策略

| 层级 | 策略 |
|------|------|
| **适配器** | 限流检测、重试、返回结构化错误 |
| **调度器** | 单厂商失败不影响其他厂商，记录失败状态 |
| **存储层** | 事务保证、冲突处理（UPSERT） |
| **API 层** | 分批导入、失败重试 |