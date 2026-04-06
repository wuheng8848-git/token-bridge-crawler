# Token Bridge Intelligence - 情报系统路线图

## 项目定位

Token Bridge 的"千里眼 & 顺风耳" - 从价格监控起步，逐步扩展为全面的市场情报系统。

**核心理念**：
- 谋定而后动：架构先行，分阶段实施
- 可扩展：新功能不破坏已有功能
- 实用主义：每个阶段都有明确价值

---

## 一、项目现状

### 1.1 当前代码审计快照

#### 已实现
- **核心抽象层**：IntelType, IntelItem, Collector 接口定义完成
- **采集器框架**：采集器注册、调度、错误处理机制实现
- **多类型采集器**：价格、API文档、用户痛点、工具生态等采集器实现
- **统一存储**：PostgreSQL 情报存储（intelligence_items 表）
- **报告生成**：日报生成器实现
- **营销信号模型**：完整的信号检测和动作生成系统
- **翻译服务**：OpenRouter 翻译服务集成
- **信号持久化**：customer_signals 表实现
- **动作持久化**：marketing_actions 表实现

#### 已闭环
- **信号/动作存储**：已创建专门的数据库表并接入主流程
- **新语义字段**：template_id, auto_execute, customer_stage, qualified_score 已实现
- **内部建议层**：所有动作统一为内部建议，不自动外发

#### 待后续迭代
- **持续运行模式**：决策层在持续运行模式下的调度策略
- **动作执行器**：人工审核后的执行机制
- **效果追踪**：动作执行效果的追踪和分析

### 1.2 已完成功能

| 模块 | 功能 | 状态 |
|------|------|------|
| 价格抓取 | Google Gemini 价格监控 | ✅ 可用 |
| 价格抓取 | OpenAI 价格监控 | ✅ 可用 |
| 价格抓取 | Anthropic 价格监控 | ✅ 可用 |
| 价格抓取 | OpenRouter 市场情报 | ✅ 可用 |
| API文档 | OpenAI API文档监控 | ✅ 可用 |
| 用户痛点 | HackerNews/Reddit 监控 | ✅ 可用 |
| 用户痛点 | 配置痛点采集 | ✅ 可用 |
| 工具生态 | 工具生态采集 | ✅ 可用 |
| 数据存储 | PostgreSQL 情报存储 | ✅ 可用 |
| 定时调度 | Cron 定时任务 | ✅ 可用 |
| 报告生成 | AI 日报 + 邮件发送 | ✅ 可用 |
| 营销信号 | 信号检测和动作生成 | ✅ 可用 |
| 信号存储 | customer_signals 表 | ✅ 可用 |
| 动作存储 | marketing_actions 表 | ✅ 可用 |
| 翻译服务 | OpenRouter 翻译 | ✅ 可用 |
| 噪声过滤 | 规则引擎（21条规则，数据库持久化） | ✅ 可用 |
| 噪声过滤 | 质量评分字段持久化 | ✅ 可用 |
| 噪声过滤 | 翻译策略优化（保守模式） | ✅ 可用 |
| 搜索引擎 | Tavily 采集器（支持迁移意愿检测） | ✅ 可用 |
| 数据源优化 | 停用低质量采集器（StackExchange/OpenAI Community） | ✅ 2026-04-06 |
| 数据源优化 | Tavily 搜索策略优化（基于官方最佳实践） | ✅ 2026-04-06 |
| 规则引擎 | 信号类型判断逻辑修复 | ✅ 2026-04-06 |
| 规则引擎 | 规则名称英文命名（避免编码问题） | ✅ 2026-04-06 |
| 处理层 | Pipeline 处理管道 | ✅ 可用 |
| 处理层 | 质量评分算法（关键词30%+影响力20%+完整度20%+时效性15%+相关性15%） | ✅ 可用 |
| 情感分析 | 情感词典分析器 | ✅ 可用 |
| 用户画像 | 客户分级（A/B/C/D 四级） | ✅ 可用 |
| HTTP客户端 | 请求重试、速率限制、User-Agent 管理 | ✅ 可用 |

### 1.3 数据表结构

```
┌─────────────────────────────────────────────────────────────────┐
│                        数据表关系图                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  intelligence_items ─────────┬─────────► customer_signals      │
│  (情报项)                     │           (客户信号)             │
│  ├─ quality_score            │                                 │
│  ├─ is_noise                 └─────────► marketing_actions     │
│  ├─ signal_type                          (营销动作)             │
│  ├─ customer_tier                                               │
│  └─ pain_score                                                  │
│        │                                                        │
│        │ 匹配                                                    │
│        ▼                                                        │
│  noise_rules                collector_runs                      │
│  (噪声规则)                  (采集器运行记录)                     │
│  ├─ rule_type                                                 │
│  ├─ weight                                                      │
│  └─ priority                                                    │
│                                                                 │
│  vendor_price_snapshots        alert_history                    │
│  (价格快照)                     (告警历史)                       │
│                                                                 │
│  vendor_price_details                                           │
│  (价格明细)                                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.4 技术债务

- [ ] 规则热更新 - 当前规则变更需要重启服务
- [ ] 迁移文件清理 - 005 和 007 迁移文件重复，需清理
- [ ] 采集器稳定性 - 部分采集器依赖网页抓取，稳定性待提升
- [ ] 多语言规则 - 中文关键词覆盖不足，需扩展
- [ ] 规则效果评估 - 缺乏误报率、漏报率监控机制
- [ ] 信号统计 API - `/api/v1/stats/signals` 功能开发中

---

## 二、架构设计

### 2.1 核心抽象层

```
┌─────────────────────────────────────────────────────────────────┐
│                        情报类型 (IntelType)                      │
├─────────────────────────────────────────────────────────────────┤
│  PRICE           - 价格情报                                      │
│  API_DOC         - API 文档变更                                  │
│  PRODUCT         - 产品发布                                      │
│  POLICY          - 政策变更                                      │
│  COMMUNITY       - 社区讨论                                      │
│  NEWS            - 行业新闻                                      │
│  USER_PAIN       - 用户痛点                                      │
│  TOOL_ECOSYSTEM  - 工具生态                                      │
│  INTEGRATION     - 集成机会                                      │
│  USER_ACQUISITION- 用户获取                                      │
│  CONVERSION      - 转化情况                                      │
│  USAGE_PATTERN   - 使用模式                                      │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 项目结构

```
token-bridge-crawler/
├── cmd/
│   ├── crawler/                    # 原有：价格爬虫入口
│   └── intelligence/               # 新增：情报系统统一入口
│
├── internal/
│   ├── core/                       # 核心抽象层
│   │   ├── types.go               # IntelType, IntelItem 定义
│   │   ├── collector.go           # Collector 接口
│   │   └── translator.go          # 翻译接口
│   │
│   ├── collectors/                 # 采集器实现
│   │   ├── price/                 # 价格采集器
│   │   ├── apidoc/                # API文档采集器
│   │   ├── policy/                # 政策采集器
│   │   ├── userpain/              # 用户痛点采集器
│   │   ├── tool/                  # 工具生态采集器
│   │   ├── search/                # Tavily 搜索引擎采集器 (新增)
│   │   ├── integration/           # 集成机会采集器
│   │   ├── useracquisition/       # 用户获取采集器
│   │   ├── conversion/            # 转化情况采集器
│   │   ├── usage/                 # 使用模式采集器
│   │   └── community/             # 社区采集器
│   │
│   ├── httpclient/                 # HTTP 客户端 (新增)
│   │   ├── client.go             # HTTP 客户端封装
│   │   ├── config.go             # 配置管理
│   │   ├── ratelimiter.go        # 速率限制
│   │   ├── retry.go              # 重试机制
│   │   └── useragent.go          # User-Agent 管理
│   │
│   ├── pipeline/                   # 处理管道 (新增)
│   │   └── pipeline.go           # Pipeline 处理流程
│   │
│   ├── processor/                  # 处理器 (新增)
│   │   ├── processor.go          # 情报处理器
│   │   └── score.go              # 质量评分算法
│   │
│   ├── rules/                      # 规则引擎 (新增)
│   │   ├── engine.go             # 规则引擎
│   │   ├── storage.go            # 规则存储
│   │   ├── types.go              # 规则类型
│   │   └── engine_test.go        # 单元测试
│   │
│   ├── sentiment/                  # 情感分析 (新增)
│   │   ├── analyzer.go           # 情感分析器
│   │   └── dict.go               # 情感词典
│   │
│   ├── profiler/                   # 用户画像 (新增)
│   │   └── profiler.go           # 客户分级
│   │
│   ├── storage/                    # 存储层
│   │   ├── storage.go             # 原有：价格存储
│   │   ├── intelligence.go        # 情报统一存储
│   │   └── translated_storage.go  # 翻译存储包装（含翻译策略）
│   │
│   ├── marketing/                  # 营销决策层
│   │   ├── types/                 # 类型定义
│   │   ├── detectors/             # 信号检测器
│   │   ├── generators/            # 动作生成器
│   │   └── signal_model.go        # 信号模型
│   │
│   ├── reporter/                   # 报告层
│   │   └── daily.go               # 日报
│   │
│   ├── scheduler/                  # 调度层
│   │   └── intelligence.go        # 情报调度
│   │
│   └── ai/                         # AI 服务层
│       ├── translator.go          # 翻译服务
│       └── translation_service.go # 翻译管理
│
├── deploy/migrations/              # 数据库迁移
│   ├── 001_create_vendor_price_tables.up.sql
│   ├── 002_create_intelligence_tables.up.sql
│   ├── 003_create_marketing_tables.up.sql
│   ├── 004_add_vendor_price_details_unique_constraint.up.sql
│   ├── 005_create_noise_rules_table.up.sql      # 规则表
│   ├── 006_add_quality_fields.up.sql            # 质量评分字段
│   └── 007_create_noise_rules_table.up.sql      # 规则表（重复，待清理）
│
└── docs/                           # 文档
```

### 2.3 数据模型

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

---

## 三、营销决策层设计

### 3.1 信号检测器 (Detectors)

| Detector | 信号类型 | 优先级 |
|----------|----------|--------|
| CostPressureDetector | cost_pressure | P0 |
| ConfigFrictionDetector | config_friction | P0 |
| ToolFragmentationDetector | tool_fragmentation | P0 |
| GovernanceStartDetector | governance_start | P1 |
| MigrationIntentDetector | migration_intent | P1 |
| GeneralInterestDetector | general_interest | P3 |

### 3.2 动作生成器 (Generators)

| Generator | 动作类型 | 输出格式 |
|-----------|----------|----------|
| CostActionGenerator | internal_note, strategy | 内部建议 |
| ConfigActionGenerator | internal_note, strategy | 内部建议 |
| ToolActionGenerator | internal_note, strategy | 内部建议 |
| GovernanceActionGenerator | short_response, technical_post | 内部建议 |
| MigrationActionGenerator | short_response, competitor_comparison, follow_up | 内部建议 |

### 3.3 语义规范

**重要**：所有生成的营销动作均为**内部建议层**，不自动外发。

| 属性 | 值 | 说明 |
|------|-----|------|
| Channel | `internal` | 统一使用内部渠道 |
| AutoExecute | `false` | 不自动执行 |
| Status | `draft` | 草稿状态，需人工审核 |

---

## 四、后续迭代方向

### 4.1 短期优化

| 任务 | 说明 | 优先级 |
|------|------|--------|
| 持续运行模式优化 | 决策层在持续运行模式下的调度策略 | P1 |
| Detector 精度优化 | 减少误判，提高信号质量 | P1 |
| 动作执行器 | 人工审核后的执行机制 | P2 |

### 4.2 中期扩展

| 方向 | 说明 | 价值 |
|------|------|------|
| 效果追踪 | 动作执行效果的追踪和分析 | 闭环反馈 |
| 信号强度优化 | 更精细的信号强度评估 | 提高精准度 |
| 渠道选择优化 | 基于信号类型的智能渠道建议 | 提高转化率 |

---

## 五、总结

### 5.1 当前里程碑

- ✅ 情报采集框架完成
- ✅ 多类型采集器实现
- ✅ 统一存储架构完成
- ✅ 营销信号模型完成
- ✅ 信号/动作持久化完成
- ✅ 内部建议层语义统一

### 5.2 核心原则

1. **架构先行**：先做好抽象层，再实现具体功能
2. **不冲突扩展**：新功能通过新增 Collector 实现，不影响已有功能
3. **分阶段交付**：每个阶段都有独立价值
4. **实用主义**：不做过度设计，按需扩展
5. **安全边界**：营销动作停留在内部建议层，不自动外发

---

**文档维护**：Token Bridge Team
**最后更新**：2026-03-31
**版本**：v2.0
