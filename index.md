# Token Bridge Intelligence - 项目索引

> **快速入口**：本文档为项目总览，帮助新成员（人类或AI助手）快速理解项目定位与结构。

---

## 一、项目定位

**Token Bridge Intelligence** 是 Token Bridge 主项目的**独立情报感知子系统**。

```
核心定位：Token Bridge 的"千里眼 & 顺风耳"

从价格监控起步 → 扩展为全面的市场情报系统 → 支撑营销决策
```

### 与主项目关系

```
token-bridge-v2/              # TB 主项目（业务 API + Admin）
    ↑
    │ API 调用
    │
token-bridge-crawler/         # 本项目（独立部署）
    │
    └── POST /v1/admin/supplier_catalog_staging/import
```

**独立性优势**：
- 解耦部署：爬虫故障不影响主业务
- 独立扩展：可单独增加厂商适配器
- 数据隔离：历史数据本地存储

---

## 二、双入口设计

| 入口 | 职责 | 状态 |
|------|------|------|
| `cmd/crawler/main.go` | 旧价格爬虫入口 | 维护中 |
| **`cmd/intelligence/main.go`** | **情报系统主干入口** | 主流程 |

**重要**：项目主干是 `cmd/intelligence/main.go`，不是旧爬虫。

---

## 三、核心功能

### 3.1 情报采集（7类）

| 类型 | 采集器 | 数据源 |
|------|--------|--------|
| 价格情报 | `price/google.go` | Google Gemini 定价页 |
| 价格情报 | `price/openai.go` | OpenAI 定价页 |
| 价格情报 | `price/anthropic.go` | Anthropic 定价页 |
| 市场情报 | `price/openrouter.go` | OpenRouter 市场 |
| 用户痛点 | `userpain/hackernews.go` | HackerNews |
| 用户痛点 | `userpain/reddit.go` | Reddit |
| 配置痛点 | `userpain/configpain.go` | 预定义痛点库 |
| 工具生态 | `tool/ecosystem.go` | 工具生态数据 |

### 3.2 营销决策层

```
情报 → 信号检测（6种）→ 动作生成（5种）→ 内部建议（人工审核）
```

**信号类型**：
| 信号 | 优先级 | 说明 |
|------|--------|------|
| `cost_pressure` | P0 | 成本压力、账单抱怨 |
| `config_friction` | P0 | 配置困难、接入门槛 |
| `tool_fragmentation` | P0 | 多工具切换成本 |
| `governance_start` | P1 | 团队预算、配额需求 |
| `migration_intent` | P1 | 竞品比较、迁移意愿 |
| `general_interest` | P3 | 泛兴趣、技术讨论 |

### 3.3 翻译服务

**编排顺序**（按免费额度和性能）：
```
火山引擎（200万字符/月，172ms）→ 百度大模型（100万字符/月，645ms）→ 百度通用（5万字符/月，1.04s）
```

**重要边界**：
```
⚠️ 所有营销动作为"内部建议层"，不自动外发

Channel: internal
AutoExecute: false
Status: draft
```

---

## 四、数据流向

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  外部数据源  │────►│  采集器     │────►│  情报存储   │
│  (API/网页) │     │ (Collectors)│     │ (PostgreSQL)│
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
                                              ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  内部建议   │◄────│  动作生成   │◄────│  信号检测   │
│ (人工审核)  │     │ (Generators)│     │ (Detectors) │
└─────────────┘     └─────────────┘     └─────────────┘
```

---

## 五、数据表结构

| 表名 | 用途 |
|------|------|
| `intelligence_items` | 情报主表（770条，100%翻译覆盖） |
| `customer_signals` | 客户信号表 |
| `marketing_actions` | 营销动作表（内部建议） |
| `vendor_price_snapshots` | 价格快照历史 |
| `vendor_price_details` | 价格明细 |
| `collector_runs` | 采集器运行日志 |

---

## 六、文档索引

| 文档 | 内容 | 优先级 |
|------|------|--------|
| [README.md](./README.md) | 快速开始、部署说明 | ⭐⭐⭐ |
| [CHANGELOG.md](./CHANGELOG.md) | 版本更新记录 | ⭐⭐ |
| [docs/DOCUMENTATION.md](./docs/DOCUMENTATION.md) | **文档规范** | ⭐⭐ |
| [docs/00-quick-reference.md](./docs/00-quick-reference.md) | 速查表（命令、配置、查询） | ⭐⭐⭐ |
| [docs/01-overview.md](./docs/01-overview.md) | 项目概述、定位 | ⭐⭐⭐ |
| [docs/02-architecture.md](./docs/02-architecture.md) | 架构设计、模块说明 | ⭐⭐ |
| [docs/03-development.md](./docs/03-development.md) | 本地开发指南 | ⭐⭐ |
| [docs/04-deployment.md](./docs/04-deployment.md) | 生产部署 | ⭐ |
| [docs/05-api.md](./docs/05-api.md) | API 接口规范 + **数据存储现状** | ⭐⭐ |
| [docs/06-roadmap.md](./docs/06-roadmap.md) | 功能路线图 | ⭐ |
| [docs/07-customer-signal-model.md](./docs/07-customer-signal-model.md) | 信号检测规则 | ⭐⭐ |

---

## 七、关键配置

### 环境变量（`.env`）

```bash
# 数据库
CRAWLER_DATABASE_URL=postgres://tbv2:tbv2_password@localhost:15432/token_bridge_crawler?sslmode=disable

# 翻译服务
BAIDU_APP_ID=xxx
BAIDU_API_KEY=xxx          # 大模型翻译
BAIDU_APP_SECRET=xxx       # 通用翻译
VOLCENGINE_ACCESS_KEY_ID=xxx
VOLCENGINE_SECRET_ACCESS_KEY=xxx

# 主项目推送
TB_BASE_URL=http://localhost:3000
TB_ADMIN_API_TOKEN=xxx
```

### 运行命令

```bash
# 单次执行（测试）
go run ./cmd/intelligence -once

# 启动定时服务
go run ./cmd/intelligence

# 编译
go build -o intelligence ./cmd/intelligence
```

---

## 八、当前状态

| 指标 | 数值 |
|------|------|
| 情报总数 | 770 条 |
| 翻译覆盖 | 100% |
| 厂商覆盖 | Google、OpenAI、Anthropic、OpenRouter |
| 信号检测器 | 6 种 |
| 动作生成器 | 5 种 |

---

## 九、快速理解清单

新成员（人类或AI）阅读顺序：

1. ✅ **本文档** - 了解项目定位
2. ✅ **[docs/00-quick-reference.md](./docs/00-quick-reference.md)** - 速查表（命令、配置、查询）
3. ✅ **README.md** - 快速启动
4. ✅ **docs/02-architecture.md** - 理解架构
5. ✅ **docs/07-customer-signal-model.md** - 理解营销决策逻辑

---

**文档维护**：Token Bridge Team  
**最后更新**：2026-03-31  
**版本**：v2.0