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
| 噪声过滤 | 规则引擎（41条规则） | ✅ 可用 |
| 噪声过滤 | 质量评分字段持久化 | ✅ 可用 |
| 噪声过滤 | 翻译策略优化（保守模式） | ✅ 可用 |
| 搜索引擎 | Tavily 采集器 | ✅ 可用 |

### 1.3 数据表结构

```
┌─────────────────────────────────────────────────────────────────┐
│                        数据表关系图                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  intelligence_items ─────────┬─────────► customer_signals      │
│  (情报项)                     │           (客户信号)             │
│                               │                                 │
│                               └─────────► marketing_actions     │
│                                           (营销动作)             │
│                                                                 │
│  collector_runs                alert_history                    │
│  (采集器运行记录)               (告警历史)                       │
│                                                                 │
│  vendor_price_snapshots        vendor_price_details             │
│  (价格快照)                     (价格明细)                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.4 技术债务

- 采集器依赖网页抓取，稳定性待提升
- 持续运行模式下决策层的调度策略待优化
- 部分 Detector 识别逻辑可进一步优化

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
│   │   └── registry.go            # 采集器注册表
│   │
│   ├── collectors/                 # 采集器实现
│   │   ├── price/                 # 价格采集器
│   │   ├── apidoc/                # API文档采集器
│   │   ├── policy/                # 政策采集器
│   │   ├── userpain/              # 用户痛点采集器
│   │   ├── tool/                  # 工具生态采集器
│   │   ├── integration/           # 集成机会采集器
│   │   ├── useracquisition/       # 用户获取采集器
│   │   ├── conversion/            # 转化情况采集器
│   │   ├── usage/                 # 使用模式采集器
│   │   └── community/             # 社区采集器
│   │
│   ├── storage/                    # 存储层
│   │   ├── storage.go             # 原有：价格存储
│   │   ├── intelligence.go        # 情报统一存储
│   │   └── translated_storage.go  # 翻译存储包装
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
│   └── 003_create_marketing_tables.up.sql
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
