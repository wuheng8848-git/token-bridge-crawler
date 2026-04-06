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
│                        Token Bridge Intelligence V2                         │
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
│         │                                                                   │
│         ▼                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           collectors                               │   │
│  │  (多类型采集器：HN/Reddit/Discord/Tavily搜索)                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     processor (新增：处理层)                         │   │
│  │  ├─ noise_filter.go      噪声过滤                                   │   │
│  │  ├─ quality_scorer.go    质量评分                                   │   │
│  │  └─ dedup.go             去重处理                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                         ┌──────────┴──────────┐                             │
│                         ▼                     ▼                             │
│  ┌──────────────────────────┐    ┌──────────────────────────┐              │
│  │     rules (新增)          │    │    profiler (新增)        │              │
│  │  ├─ engine.go  规则引擎    │    │  ├─ fetcher.go  画像获取  │              │
│  │  ├─ loader.go  规则加载    │    │  ├─ tier.go     客户分级  │              │
│  │  └─ types.go   类型定义    │    │  └─ cache.go    画像缓存  │              │
│  └──────────────────────────┘    └──────────────────────────┘              │
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
| **AI搜索** | `search/tavily.go` | Tavily AI 搜索采集 | 🆕 **新增** |

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

### 4. 处理层 (internal/processor/) - 新增

| 文件 | 功能 | 状态 |
|------|------|------|
| `processor.go` | 处理器接口定义 | 🆕 待开发 |
| `noise_filter.go` | 噪声过滤 | 🆕 待开发 |
| `quality_scorer.go` | 质量评分 | 🆕 待开发 |
| `dedup.go` | 去重处理 | 🆕 待开发 |

**处理流程**:
```
原始情报 → 噪声判断 → 质量评分 → 过滤决策
    │          │          │          │
    │      噪声?       0-100分    入库/丢弃
    │          │          │
    │          ▼          ▼
    │      直接丢弃    分级处理
```

**处理器接口**:
```go
type Processor interface {
    // 处理情报，返回处理结果
    Process(ctx context.Context, item core.IntelItem) ProcessResult
}

type ProcessResult struct {
    IsNoise      bool     // 是否噪声
    QualityScore int      // 质量评分 (0-100)
    FilterReason string   // 过滤原因
    Keep         bool     // 是否保留
}
```

### 5. 规则引擎层 (internal/rules/) - 新增

| 文件 | 功能 | 状态 |
|------|------|------|
| `engine.go` | 规则引擎核心 | 🆕 待开发 |
| `loader.go` | 规则加载（支持热更新） | 🆕 待开发 |
| `types.go` | 规则类型定义 | 🆕 待开发 |

**功能**:
- **规则配置化**：规则存储在数据库，无需改代码
- **热更新**：修改规则后自动生效，无需重启
- **权重机制**：正数=噪声特征，负数=信号特征

**规则引擎接口**:
```go
type RuleEngine interface {
    // 加载所有激活的规则
    LoadRules(ctx context.Context) error

    // 评估情报是否为噪声
    Evaluate(item *core.IntelItem) (isNoise bool, score int, reasons []string)

    // 重新加载规则（热更新）
    Reload(ctx context.Context) error
}
```

### 6. 用户画像层 (internal/profiler/) - 新增

| 文件 | 功能 | 状态 |
|------|------|------|
| `fetcher.go` | 用户画像获取 | 🆕 待开发 |
| `tier.go` | 客户价值分级 | 🆕 待开发 |
| `cache.go` | 画像缓存 | 🆕 待开发 |

**功能**:
- **用户画像获取**：从社区 API 获取发帖人信息
- **客户价值分级**：S/A/B/C 四级
- **画像缓存**：避免重复请求

**用户画像接口**:
```go
type Profiler interface {
    // 获取用户画像
    FetchProfile(ctx context.Context, platform, userID string) (*UserProfile, error)

    // 判断客户价值等级
    ClassifyTier(profile *UserProfile) CustomerTier
}

type CustomerTier string

const (
    TierS CustomerTier = "S"  // 立即回帖引流
    TierA CustomerTier = "A"  // 重点关注
    TierB CustomerTier = "B"  // 一般关注
    TierC CustomerTier = "C"  // 低优先级
)
```

### 7. 存储层 (internal/storage/)

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

### 8. 营销决策层 (internal/marketing/)

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

### 9. 报告层 (internal/reporter/)

| 文件 | 功能 | 状态 |
|------|------|------|
| `daily.go` | 日报生成器 | ✅ 可用 |

**功能**:
- **日报生成**：自动生成每日情报摘要
- **邮件发送**：支持邮件通知

### 10. AI 服务层 (internal/ai/)

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

### V2 执行链（情报系统 + 处理层 + 决策层）

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  外部数据源  │────►│  采集器     │────►│   处理器    │
│  网站/API   │     │ HN/Reddit   │     │ 噪声过滤    │
│  Tavily搜索  │     │ Discord     │     │ 质量评分    │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                    ┌──────────────────────────┴──────────┐
                    ▼                                     ▼
            ┌─────────────┐                      ┌─────────────┐
            │  规则引擎    │                      │  用户画像   │
            │ 噪声规则    │                      │ 客户分级    │
            │ 热更新配置   │                      │ S/A/B/C    │
            └─────────────┘                      └─────────────┘
                    │                                     │
                    └──────────────────┬──────────────────┘
                                       ▼
                              ┌─────────────┐
                              │   存储层    │
                              │ 过滤决策    │
                              │ 优质入库    │
                              └──────┬──────┘
                                     │
                                     ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  报告生成   │◄────│  调度器     │◄────│  情报处理   │
│  (日报)     │     │ (定时任务)  │     │ 翻译/总结   │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  动作存储   │◄────│  动作生成   │◄────│  信号检测   │
│ (内部建议)  │     │ (内部建议)  │     │ (客户信号)  │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                                               ▼
                              ┌─────────────────────────┐
                              │      Admin 后台         │
                              │  优质信号展示 + 用户画像 │
                              │  规则配置 + 效果统计    │
                              └─────────────────────────┘
```

### 情报处理流程详解

```
情报进入
    │
    ▼
┌─────────────────────────────────────┐
│ 第一步：噪声判断（规则引擎）          │
│                                     │
│ 是否为噪声？                         │
│ └─ 包含营销关键词？                  │
│ └─ 内容长度 < 20 字符？              │
│ └─ 与 AI API 完全无关？              │
└─────────────────────────────────────┘
    │
    ├── 是噪声 ──────────► 直接丢弃，不入库
    │
    ▼ 否，可能是信号
┌─────────────────────────────────────┐
│ 第二步：质量评分                     │
│                                     │
│ 质量分 = 关键词(30%) + 影响力(20%)   │
│        + 完整度(20%) + 时效性(15%)   │
│        + 相关性(15%)                 │
└─────────────────────────────────────┘
    │
    ├── 0-20 分 ───► 不入库
    ├── 20-40 分 ──► 入库，默认隐藏
    ├── 40-70 分 ──► 入库，按需获取画像
    │
    ▼ 70-100 分（高质量）
┌─────────────────────────────────────┐
│ 第三步：用户画像获取                 │
│                                     │
│ 获取发帖人信息：                     │
│ └─ Bio、技术栈                       │
│ └─ Karma、历史发帖                   │
│ └─ GitHub、项目信息                  │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ 第四步：客户价值分级                 │
│                                     │
│ 是开发者吗？ 有实际项目吗？          │
│ 有付费能力吗？ 有多个痛点吗？        │
└─────────────────────────────────────┘
    │
    ├── 符合 4 项 ──► S 级（立即回帖引流）
    ├── 符合 3 项 ──► A 级（重点关注）
    ├── 符合 2 项 ──► B 级（一般关注）
    └── 符合 0-1 项 ─► C 级（低优先级）
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
| **处理层** | 噪声过滤失败不阻塞，记录日志继续处理 |

## 目录结构变化

```
internal/
├── processor/          # 新增：处理层
│   ├── processor.go        # 处理器接口
│   ├── noise_filter.go     # 噪声过滤
│   ├── quality_scorer.go   # 质量评分
│   └── dedup.go            # 去重处理
│
├── rules/              # 新增：规则引擎
│   ├── engine.go           # 规则引擎核心
│   ├── loader.go           # 规则加载（热更新）
│   └── types.go            # 规则类型定义
│
├── profiler/           # 新增：用户画像
│   ├── fetcher.go          # 画像获取
│   ├── tier.go             # 客户分级
│   └── cache.go            # 画像缓存
│
├── collectors/         # 现有：采集器
│   └── search/             # 新增：AI搜索采集
│       └── tavily.go       # Tavily 搜索采集器
│
├── marketing/          # 现有：营销决策（改造）
│   └── detectors/          # 改造：读取规则引擎配置
│
└── storage/            # 现有：存储层（扩展）
    └── intelligence.go     # 扩展：新增质量字段、用户画像表
```
