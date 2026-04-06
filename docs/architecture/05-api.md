# Token Bridge Intelligence - API 文档

## 概述

Token Bridge Intelligence 通过 TB Admin API 与主项目交互。本文档描述情报系统使用的接口规范。

**重要**：本项目是情报感知系统，不是自动化营销系统。所有营销动作停留在内部建议层，不自动外发。

## TB API 接口

### 导入价格到 Staging

```http
POST /v1/admin/supplier_catalog_staging/import
```

**请求头**:
```http
Authorization: Bearer {TB_ADMIN_API_TOKEN}
Content-Type: application/json
```

**请求体**:
```json
[
  {
    "source": "openai-2026-03-24",
    "model_code": "gpt-4o",
    "model_name": "GPT-4o",
    "pricing_raw": {
      "input_usd_per_million": 2.5,
      "output_usd_per_million": 10.0,
      "currency": "USD",
      "price_type": "vendor_list_price",
      "schema_version": "v1",
      "captured_at": "2026-03-24T02:00:00Z"
    },
    "suggested_retail_usd_minor_per_1k": null
  }
]
```

**响应**:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "imported_count": 50,
  "errors": []
}
```

## 内部数据结构

### IntelItem（情报项）

```go
type IntelItem struct {
    ID          string                 // UUID
    IntelType   IntelType              // 情报类型
    Source      string                 // 数据源
    SourceID    string                 // 源系统ID
    Title       string                 // 标题
    Content     string                 // 内容
    URL         string                 // 原始链接
    Metadata    map[string]interface{} // 扩展数据
    CapturedAt  time.Time              // 采集时间
    PublishedAt time.Time              // 发布时间
    Status      string                 // 处理状态
}
```

### CustomerSignal（客户信号）

```go
type CustomerSignal struct {
    ID           string    // UUID
    IntelItemID  string    // 关联情报项ID
    SignalType   string    // 信号类型
    Strength     int       // 信号强度 1-3
    Content      string    // 信号内容摘要
    Platform     string    // 来源平台
    Author       string    // 作者标识
    URL          string    // 原始链接
    Metadata     JSONB     // 扩展数据
    Status       string    // new, processed, ignored
    DetectedAt   time.Time // 检测时间
}
```

### MarketingAction（营销动作）

```go
type MarketingAction struct {
    ID              string    // UUID
    ActionType      string    // 动作类型
    Channel         string    // 目标渠道（统一为 internal）
    Title           string    // 动作标题
    Content         string    // 动作内容
    TemplateID      string    // 模板ID
    TargetAudience  string    // 目标受众
    Priority        int       // 优先级 1-5
    SignalIDs       JSONB     // 关联信号ID列表
    AutoExecute     bool      // 是否自动执行（统一为 false）
    CustomerStage   string    // 客户阶段
    QualifiedScore  float64   // 资格化分数 0-100
    Metadata        JSONB     // 扩展数据
    Status          string    // draft, pending, executed, failed
    ScheduledAt     time.Time // 计划执行时间
    ExecutedAt      time.Time // 实际执行时间
}
```

## 数据库表结构

### 情报相关表

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

### 价格相关表（旧）

```sql
-- 价格快照表
CREATE TABLE vendor_price_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    snapshot_at TIMESTAMPTZ NOT NULL,
    total_models INT NOT NULL DEFAULT 0,
    new_models INT NOT NULL DEFAULT 0,
    updated_models INT NOT NULL DEFAULT 0,
    removed_models INT NOT NULL DEFAULT 0,
    raw_data_hash TEXT,
    status TEXT NOT NULL DEFAULT 'success',
    error_log TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 价格明细表
CREATE TABLE vendor_price_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID REFERENCES vendor_price_snapshots(id),
    vendor TEXT NOT NULL,
    model_code TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    input_usd_per_million NUMERIC(12, 6),
    output_usd_per_million NUMERIC(12, 6),
    currency TEXT DEFAULT 'USD',
    capabilities JSONB,
    change_type TEXT,
    prev_price JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(vendor, model_code, snapshot_date)
);
```

## 常用查询示例

### 查询情报采集状态

```sql
SELECT
    intel_type,
    source,
    status,
    items_count,
    started_at
FROM collector_runs
ORDER BY started_at DESC
LIMIT 10;
```

### 查询检测到的信号

```sql
SELECT
    signal_type,
    strength,
    platform,
    content,
    detected_at
FROM customer_signals
WHERE status = 'new'
ORDER BY detected_at DESC
LIMIT 20;
```

### 查询生成的内部建议

```sql
SELECT
    action_type,
    channel,
    title,
    priority,
    status,
    created_at
FROM marketing_actions
WHERE status = 'draft'
ORDER BY priority DESC, created_at DESC
LIMIT 20;
```

### 确认动作未自动执行

```sql
-- 所有动作应为 draft 状态
SELECT status, COUNT(*)
FROM marketing_actions
GROUP BY status;

-- auto_execute 应为 false
SELECT auto_execute, COUNT(*)
FROM marketing_actions
GROUP BY auto_execute;
```

## 配置接口

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | `config.yaml` |
| `-once` | 只执行一次，不启动定时任务 | `false` |

### 环境变量

| 变量名 | 说明 |
|--------|------|
| `CRAWLER_DATABASE_URL` | 数据库连接字符串 |
| `TB_ADMIN_API_TOKEN` | TB Admin API Token |
| `TB_BASE_URL` | TBv2 后端 API 基础地址（例如 `http://127.0.0.1:8080`），不要填写前端/控制台/开发服务器端口（如 `5173` / `3000` / `3001`） |
| `CRAWLER_AI_API_KEY` | AI 日报 API Key |
| `OPENROUTER_API_KEY` | OpenRouter 翻译服务 API Key |
| `OPENAI_API_KEY` | OpenAI API Key |
| `SMTP_HOST` | SMTP 服务器地址 |
| `SMTP_USER` | SMTP 用户名 |
| `SMTP_PASS` | SMTP 密码 |

#### TB_BASE_URL 防误配

`TB_BASE_URL` 用于拼接并调用 TBv2 Admin API（例如 `POST /v1/admin/supplier_catalog_staging/import`）。它必须指向 TBv2 的后端 API 服务端口，而不是前端页面端口。

- 正确示例：`http://127.0.0.1:8080`（TBv2 API）
- 常见误配：`http://127.0.0.1:5173` / `http://localhost:3000` / `http://localhost:3001`（通常是前端 dev server 或管理台端口）

## 采集器接口规范

```go
type Collector interface {
    Name() string                    // 采集器名称
    IntelType() core.IntelType       // 情报类型
    Source() string                  // 数据源
    Fetch(ctx context.Context) ([]core.IntelItem, error)  // 采集数据
    RateLimit() time.Duration        // 请求间隔
}
```

## 信号检测器接口规范

```go
type Detector interface {
    GetSupportedTypes() []SignalType                    // 支持的信号类型
    DetectFromIntel(item IntelItem) ([]CustomerSignal, error)  // 检测信号
}
```

## 动作生成器接口规范

```go
type Generator interface {
    GetSupportedActionTypes() []ActionType                         // 支持的动作类型
    GenerateFromSignals(signals []CustomerSignal) ([]MarketingAction, error)  // 生成动作
}
```

---

## 数据存储现状

> 最后更新：2026-03-31

### 表记录统计

| 表名 | 记录数 | 用途 |
|------|--------|------|
| `intelligence_items` | 770 | 情报主表 |
| `customer_signals` | 1381 | 客户信号 |
| `marketing_actions` | 1184 | 营销动作（内部建议） |
| `collector_runs` | 194 | 采集器运行日志 |
| `alert_rules` | 4 | 告警规则 |
| `vendor_price_details` | 22 | 价格明细（旧） |
| `vendor_price_snapshots` | 3 | 价格快照（旧） |

### 数据存储结构

```
intelligence_items 表
├── id (uuid)                    # 主键
├── intel_type (text)            # 情报类型
├── source (text)                # 数据源
├── title (text)                 # 原始标题（英文）
├── content (text)               # 原始内容（英文）
├── metadata (jsonb)             # 元数据 + 翻译数据 ⭐
│   ├── title_zh                 # 翻译后标题（中文）
│   ├── content_zh               # 翻译后内容（中文）
│   ├── model_code               # 模型代码
│   ├── model_name               # 模型名称
│   ├── input_price              # 输入价格
│   ├── output_price             # 输出价格
│   ├── currency                 # 货币
│   ├── author                   # 作者
│   ├── platform                 # 平台
│   └── ...                      # 其他扩展字段
└── created_at, updated_at       # 时间戳
```

### 翻译数据存储

| 项目 | 状态 |
|------|------|
| 翻译覆盖率 | **100%** (770/770) |
| 存储位置 | `metadata` JSONB 字段 |
| 字段名 | `title_zh`, `content_zh` |
| 存储方式 | 与元数据合并存储 |

### metadata 常见字段

| 字段 | 数量 | 说明 |
|------|------|------|
| `title_zh` | 770 | 翻译后标题 |
| `content_zh` | 770 | 翻译后内容 |
| `model_code` | 588 | 模型代码 |
| `model_name` | 588 | 模型名称 |
| `input_price` | 588 | 输入价格 |
| `output_price` | 588 | 输出价格 |
| `currency` | 588 | 货币 |
| `schema_version` | 588 | Schema 版本 |
| `price_type` | 588 | 价格类型 |
| `change_type` | 588 | 变更类型 |
| `author` | 168 | 作者 |
| `platform` | 168 | 平台 |
| `value_proposition` | 168 | 价值主张 |
| `marketing_angle` | 168 | 营销角度 |

### 常用查询

```sql
-- 查看情报统计
SELECT COUNT(*) FROM intelligence_items;

-- 查看翻译覆盖率
SELECT 
    COUNT(*) FILTER (WHERE metadata ? 'title_zh') as translated,
    COUNT(*) as total
FROM intelligence_items;

-- 查看 metadata 中的字段
SELECT key, COUNT(*) 
FROM intelligence_items, jsonb_object_keys(metadata) as key 
GROUP BY key 
ORDER BY COUNT(*) DESC;

-- 查看未处理的信号
SELECT * FROM customer_signals WHERE status = 'new' LIMIT 10;

-- 查看待审核的营销动作
SELECT * FROM marketing_actions WHERE status = 'draft' LIMIT 10;
```

---

**文档维护**：Token Bridge Team
**最后更新**：2026-03-31
**版本**：v2.0
