# Token Bridge Intelligence

Token Bridge 的**情报感知系统**，负责从多维度采集市场情报，检测客户信号，生成内部建议。

> **快速入口**：新成员请先阅读 [docs/index.md](./docs/index.md) 了解项目全貌。

---

## 项目定位

```
Token Bridge 的"千里眼 & 顺风耳"

从价格监控起步 → 扩展为全面的市场情报系统 → 支撑营销决策
```

**重要边界**：所有营销动作为"内部建议层"，不自动外发。

---

## 功能特性

### 情报采集（7类）

| 类型 | 数据源 | 说明 |
|------|--------|------|
| 价格情报 | Google、OpenAI、Anthropic、OpenRouter | 厂商刊例价监控 |
| 用户痛点 | HackerNews、Reddit | 用户抱怨、需求讨论 |
| 配置痛点 | 预定义痛点库 | API配置、接入门槛 |
| 工具生态 | 工具生态数据 | 多工具使用情况 |

### 营销决策层

```
情报 → 信号检测（6种）→ 动作生成（5种）→ 内部建议（人工审核）
```

**信号类型**：成本压力(P0)、配置摩擦(P0)、工具碎片化(P0)、治理起点(P1)、迁移意愿(P1)、泛兴趣(P3)

### 翻译服务

**编排顺序**（按免费额度和性能）：
```
火山引擎（200万字符/月，172ms）→ 百度大模型（100万字符/月）→ 百度通用（5万字符/月）
```

### 其他特性

- **定时调度**：Cron 表达式配置，默认每日凌晨 2 点
- **限流降级**：自动检测限流，失败后指数退避
- **历史版本**：保存每次抓取快照，支持趋势分析
- **AI 日报**：自动生成总结并邮件发送

---

## 项目结构

```
token-bridge-crawler/
├── cmd/
│   ├── intelligence/           # 情报系统主干入口 ⭐
│   │   └── main.go
│   └── crawler/                # 旧价格爬虫入口（维护中）
│       └── main.go
│
├── internal/
│   ├── core/                   # 核心抽象层
│   │   ├── types.go           # IntelType, IntelItem 定义
│   │   └── collector.go       # Collector 接口
│   │
│   ├── collectors/             # 采集器实现
│   │   ├── price/             # 价格采集器
│   │   ├── userpain/          # 用户痛点采集器
│   │   └── tool/              # 工具生态采集器
│   │
│   ├── marketing/              # 营销决策层
│   │   ├── detectors/         # 信号检测器（6种）
│   │   ├── generators/        # 动作生成器（5种）
│   │   └── signal_model.go    # 信号模型
│   │
│   ├── storage/                # 存储层
│   │   ├── intelligence.go    # 情报存储
│   │   └── translated_storage.go
│   │
│   ├── ai/                     # AI 服务
│   │   ├── translator.go      # 翻译服务
│   │   └── translation_service.go
│   │
│   └── scheduler/              # 调度层
│       └── intelligence.go
│
├── docs/                       # 文档
│   ├── 01-overview.md
│   ├── 02-architecture.md
│   └── ...
│
├── index.md                    # 项目索引（快速入口）
├── config.yaml                 # 配置文件
├── .env.example                # 环境变量示例
└── Dockerfile                  # 容器镜像
```

---

## 快速开始

### 1. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 填入必要配置：

```env
# 数据库
CRAWLER_DATABASE_URL=postgres://tbv2:tbv2_password@localhost:15432/token_bridge_crawler?sslmode=disable

# 翻译服务
BAIDU_APP_ID=xxx
BAIDU_API_KEY=xxx
BAIDU_APP_SECRET=xxx
VOLCENGINE_ACCESS_KEY_ID=xxx
VOLCENGINE_SECRET_ACCESS_KEY=xxx

# 主项目推送（可选）
TB_BASE_URL=http://127.0.0.1:8080
TB_ADMIN_API_TOKEN=xxx
```

### 2. 本地运行

```bash
# 安装依赖
go mod download

# 单次执行（测试）
go run ./cmd/intelligence -once

# 启动定时服务
go run ./cmd/intelligence

# 编译
go build -o intelligence ./cmd/intelligence
```

### 3. Docker 运行

```bash
docker build -t token-bridge-intelligence .
docker run -d --name intelligence \
  -v $(pwd)/config.yaml:/root/config.yaml \
  -e CRAWLER_DATABASE_URL=postgres://... \
  token-bridge-intelligence
```

---

## 数据表

| 表名 | 用途 |
|------|------|
| `intelligence_items` | 情报主表 |
| `customer_signals` | 客户信号表 |
| `marketing_actions` | 营销动作表（内部建议） |
| `vendor_price_snapshots` | 价格快照历史 |
| `vendor_price_details` | 价格明细 |

---

## 与 TB 主项目的关系

```
token-bridge-v2/              # TB 主项目（业务 API + Admin）
    ↑
    │ API 调用
    │
token-bridge-crawler/         # 本项目（独立部署）
    │
    └── POST /v1/admin/supplier_catalog_staging/import
```

- **独立部署**：情报系统作为独立服务运行
- **API 通信**：通过 TB Admin API 导入数据
- **数据库**：可与 TB 共用，也可独立部署
- **目录导入带（HTTP）**：通过 `TB_BASE_URL` 指向 TBv2 API（示例 `http://127.0.0.1:8080`），使用 `TB_ADMIN_API_TOKEN` 调用 `POST /v1/admin/supplier_catalog_staging/import` 导入目录/价格到 staging
- **情报聚合带（DB）**：采集到的情报、翻译结果、信号、内部建议落地到 `CRAWLER_DATABASE_URL` 对应的 PostgreSQL；TBv2/BI 可按“共库/独立库”口径做聚合分析与运营看板

---

## 文档索引

| 文档 | 内容 |
|------|------|
| [index.md](./index.md) | 项目总览（推荐入口） |
| [CHANGELOG.md](./CHANGELOG.md) | 版本更新记录 |
| [docs/00-quick-reference.md](./docs/00-quick-reference.md) | 速查表（命令、配置、查询） |
| [docs/01-overview.md](./docs/01-overview.md) | 项目概述 |
| [docs/02-architecture.md](./docs/02-architecture.md) | 架构设计 |
| [docs/03-development.md](./docs/03-development.md) | 开发指南 |
| [docs/07-customer-signal-model.md](./docs/07-customer-signal-model.md) | 信号检测规则 |

---

## 当前状态

| 指标 | 数值 |
|------|------|
| 情报总数 | 770 条 |
| 翻译覆盖 | 100% |
| 厂商覆盖 | Google、OpenAI、Anthropic、OpenRouter |
| 信号检测器 | 6 种 |
| 动作生成器 | 5 种 |

---

## License

MIT
