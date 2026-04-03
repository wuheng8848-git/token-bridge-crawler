# 客户信号模型与动作触发规则

## 1. 当前实现状态

### 1.1 已实现的 Detectors

| Detector 名称 | 信号类型 | 功能说明 | 实现状态 |
|--------------|---------|----------|----------|
| CostPressureDetector | 成本压力信号 | 检测用户关于成本、账单、预算的抱怨 | ✅ 已实现 |
| ConfigFrictionDetector | 配置摩擦信号 | 检测用户关于API配置、provider设置的困难 | ✅ 已实现 |
| ToolFragmentationDetector | 工具碎片化信号 | 检测用户同时使用多个AI工具的情况 | ✅ 已实现 |
| GovernanceStartDetector | 治理起点信号 | 检测用户关于团队预算、配额、分账的需求 | ✅ 已实现 |
| MigrationIntentDetector | 迁移意愿信号 | 检测用户比较竞品、考虑替代方案的意图 | ✅ 已实现 |
| GeneralInterestDetector | 泛兴趣信号 | 检测用户对AI的一般兴趣 | ✅ 已实现 |

### 1.2 已实现的 Generators

| Generator 名称 | 动作类型 | 功能说明 | 实现状态 |
|---------------|----------|----------|----------|
| CostActionGenerator | 成本相关动作 | 生成成本对比、8.5x节省等内部建议 | ✅ 已实现 |
| ConfigActionGenerator | 配置相关动作 | 生成接入教程、配置清单等内部建议 | ✅ 已实现 |
| ToolActionGenerator | 工具相关动作 | 生成统一接入、策略层等内部建议 | ✅ 已实现 |
| GovernanceActionGenerator | 治理相关动作 | 生成透明计费、团队控制等内部建议 | ✅ 已实现 |
| MigrationActionGenerator | 迁移相关动作 | 生成竞品对比、迁移理由等内部建议 | ✅ 已实现 |

### 1.3 信号模型状态

- **当前状态**：已完整实现并接入主流程
- **接入情况**：在单次执行模式（-once）下完整运行
- **存储情况**：信号和动作已持久化到数据库
- **语义定位**：所有动作为**内部建议层**，不自动外发

### 1.4 数据存储结构

#### customer_signals 表

```sql
CREATE TABLE customer_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intel_item_id UUID REFERENCES intelligence_items(id),
    signal_type TEXT NOT NULL,           -- 信号类型
    strength INT NOT NULL DEFAULT 1,     -- 信号强度 1-3
    content TEXT,                        -- 信号内容摘要
    platform TEXT,                       -- 来源平台
    author TEXT,                         -- 作者标识
    url TEXT,                            -- 原始链接
    metadata JSONB DEFAULT '{}',         -- 扩展数据
    status TEXT DEFAULT 'new',           -- new, processed, ignored
    detected_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### marketing_actions 表

```sql
CREATE TABLE marketing_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_type TEXT NOT NULL,           -- 动作类型
    channel TEXT NOT NULL,               -- 目标渠道
    title TEXT,                          -- 动作标题
    content TEXT,                        -- 动作内容
    template_id TEXT,                    -- 模板ID（可选）
    target_audience TEXT,                -- 目标受众
    priority INT NOT NULL DEFAULT 3,     -- 优先级 1-5
    signal_ids JSONB DEFAULT '[]',       -- 关联信号ID
    auto_execute BOOLEAN DEFAULT FALSE,  -- 是否自动执行
    customer_stage TEXT,                 -- 客户阶段
    qualified_score NUMERIC(5,2),        -- 资格化分数 0-100
    metadata JSONB DEFAULT '{}',         -- 扩展数据
    status TEXT DEFAULT 'draft',         -- draft, pending, executed, failed
    scheduled_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 2. 客户信号模型

### 2.1 核心客户画像
- **目标群体**：美国付费型独立开发者与小团队
- **特征**：
  - 有稳定的AI API使用需求
  - 对成本敏感，关注token使用效率
  - 技术能力强，但配置管理时间有限
  - 追求开发效率和系统稳定性

### 2.2 优先识别的信号

#### 2.2.1 成本压力信号 (P0)
- **信号类型**：cost_pressure
- **具体信号**：
  - 账单变高、超预算
  - 订阅不够用、订阅不划算
  - API比订阅更省

#### 2.2.2 配置摩擦信号 (P0)
- **信号类型**：config_friction
- **具体信号**：
  - 第三方API难接、教程难找
  - provider / base URL / key 配置麻烦
  - MCP、skills配置复杂

#### 2.2.3 工具碎片化信号 (P0)
- **信号类型**：tool_fragmentation
- **具体信号**：
  - 同时用多个AI工具、多个套餐
  - 多个模型供应商
  - 工具间切换成本高

#### 2.2.4 治理起点信号 (P1)
- **信号类型**：governance_start
- **具体信号**：
  - 团队里谁在烧token不清楚
  - 需要预算/配额/分账
  - 多人协作导致的管理问题

#### 2.2.5 迁移意愿信号 (P1)
- **信号类型**：migration_intent
- **具体信号**：
  - 公开比较模型、比较订阅与API
  - 讨论替代OpenRouter / 单一供应商
  - 寻求更灵活的解决方案

#### 2.2.6 泛兴趣信号 (P3)
- **信号类型**：general_interest
- **具体信号**：
  - 讨论AI很酷、模型榜单
  - 泛体验分享
  - 技术趋势讨论

## 3. 动作触发规则

### 3.1 信号判断规则

| 规则 | 判断 | 优先级 |
|------|------|--------|
| 同时出现**成本压力 + 配置摩擦** | 视为高价值线索，优先进入后续动作层 | P0 |
| 同时出现**工具碎片化 + 迁移意愿** | 视为适合"统一接入 / 策略层"叙事 | P0 |
| 出现**治理起点** | 视为小团队或中型团队信号，后续话术从"省钱"切到"可控" | P1 |
| 只有**泛兴趣信号**、无成本或配置表述 | 仅保留观察，不进入重点跟进队列 | P3 |

### 3.2 动作输出规范

**重要**：所有生成的动作均为**内部建议层**，不自动外发。

| 属性 | 值 | 说明 |
|------|-----|------|
| Channel | `internal` | 统一使用内部渠道 |
| AutoExecute | `false` | 不自动执行 |
| Status | `draft` | 草稿状态，需人工审核 |

### 3.3 动作类型与适用场景

| 动作类型 | 适用场景 | 目标 |
|----------|----------|------|
| **内部备注** (internal_note) | 记录信号响应建议 | 供团队参考 |
| **策略建议** (strategy) | 内容创作方向建议 | 指导营销策略 |
| **短内容回应** (short_response) | 社区讨论响应建议 | 获取互动、验证信号 |
| **技术文章** (technical_post) | 成本、路由等高热主题 | 获取高质量流量 |
| **配置教程** (setup_guide) | 存在接入门槛或配置痛点 | 降低转化摩擦 |
| **竞品对比** (competitor_comparison) | 用户已在比较方案 | 推动迁移意愿 |

## 4. 新语义字段说明

### 4.1 MarketingAction 新增字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `template_id` | TEXT | 模板ID，用于关联预定义的内容模板 |
| `auto_execute` | BOOLEAN | 是否自动执行，当前统一为 `false` |
| `customer_stage` | TEXT | 客户阶段：awareness / consideration / decision / retention |
| `qualified_score` | NUMERIC(5,2) | 资格化分数（0-100），由 SignalQualifier 计算 |

### 4.2 客户阶段映射

| 信号类型 | 客户阶段 | 说明 |
|----------|----------|------|
| general_interest | awareness | 认知阶段 |
| cost_pressure | consideration | 考虑阶段 |
| config_friction | consideration | 考虑阶段 |
| tool_fragmentation | consideration | 考虑阶段 |
| migration_intent | decision | 决策阶段 |
| governance_start | retention | 留存阶段 |

## 5. 执行流程

```
┌─────────────────────────────────────────────────────────────────┐
│                     情报采集 → 信号检测 → 动作生成                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. 采集器采集情报                                               │
│     ↓                                                           │
│  2. 存储到 intelligence_items 表                                │
│     ↓                                                           │
│  3. Detectors 检测信号                                          │
│     ↓                                                           │
│  4. SignalQualifier 评估信号资格                                │
│     ↓                                                           │
│  5. 存储到 customer_signals 表                                  │
│     ↓                                                           │
│  6. Generators 生成内部建议动作                                  │
│     ↓                                                           │
│  7. 存储到 marketing_actions 表                                 │
│     ↓                                                           │
│  8. 人工审核后执行（非自动）                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 6. 当前状态总结

### 6.1 已完成

- ✅ 6 种信号检测器实现
- ✅ 5 种动作生成器实现
- ✅ 信号资格评估器实现
- ✅ 信号持久化存储（customer_signals 表）
- ✅ 动作持久化存储（marketing_actions 表）
- ✅ 新语义字段支持（template_id, auto_execute, customer_stage, qualified_score）
- ✅ 内部建议层语义统一

### 6.2 待后续迭代

- 持续运行模式下的营销信号处理
- 动作执行器（人工审核后执行）
- 动作效果追踪
- Detector 识别精度优化

---

**文档维护**：Token Bridge Team
**最后更新**：2026-03-31
**版本**：v2.0