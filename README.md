# Token Bridge Crawler

Token Bridge 厂商刊例价抓取服务，用于自动抓取 Google/OpenAI/Anthropic 等厂商的模型价格，并导入到 TB 系统中。

## 功能特性

- **多厂商支持**: Google (Gemini)、OpenAI、Anthropic
- **定时抓取**: 支持 Cron 表达式配置，默认每日凌晨 2 点
- **限流降级**: 自动检测限流，失败后指数退避（日频→周频）
- **历史版本**: 保存每次抓取的历史，支持价格趋势分析
- **AI 日报**: 自动生成 AI 总结并邮件发送给运营团队
- **独立部署**: 独立项目，通过 API 与 TB 交互

## 项目结构

```
token-bridge-crawler/
├── cmd/crawler/           # 主入口
│   └── main.go
├── internal/
│   ├── adapters/          # 厂商适配器
│   │   ├── google.go      # Google Gemini
│   │   ├── openai.go      # OpenAI
│   │   └── anthropic.go   # Anthropic
│   ├── ai/                # AI 日报生成
│   ├── mail/              # 邮件发送
│   ├── storage/           # 历史数据存储
│   ├── scheduler.go       # 调度器
│   └── tbclient.go        # TB API 客户端
├── config.yaml            # 配置文件
├── .env.example           # 环境变量示例
├── Dockerfile             # 容器镜像
└── README.md
```

## 快速开始

### 1. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 填入必要配置
```

### 2. 运行数据库迁移

需要先创建历史表：

```sql
-- vendor_price_snapshots
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

-- vendor_price_details
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
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snapshots_vendor_date ON vendor_price_snapshots(vendor, snapshot_date DESC);
CREATE INDEX idx_details_lookup ON vendor_price_details(vendor, model_code, snapshot_date DESC);
```

### 3. 本地运行

```bash
# 安装依赖
go mod download

# 单次执行（测试）
go run ./cmd/crawler -once

# 启动定时服务
go run ./cmd/crawler
```

### 4. Docker 运行

```bash
# 构建镜像
docker build -t token-bridge-crawler .

# 运行
docker run -d \
  --name crawler \
  -v $(pwd)/config.yaml:/root/config.yaml \
  -e CRAWLER_DATABASE_URL=postgres://... \
  token-bridge-crawler
```

## 配置说明

### 抓取频率

- **正常**: 每日一次（cron: `0 2 * * *`）
- **限流降级**: 连续失败后自动调整为 2天 → 4天 → 每周

### 厂商优先级

1. Google (Gemini) - P0
2. OpenAI - P1
3. Anthropic - P2

### 邮件日报

- 抓取完成后自动生成 AI 总结
- 包含统计数据和 CSV 附件
- 支持多收件人

## API 依赖

爬虫通过以下 TB API 交互：

```
POST /v1/admin/supplier_catalog_staging/import
```

需要 Admin API Token。

## 与 TB 项目的关系

```
token-bridge-v2/           # 主项目（业务 API + Admin）
├── apps/api/             # TB API 服务
└── ...

token-bridge-crawler/      # 爬虫项目（当前）
├── cmd/crawler/
└── ...
```

爬虫独立部署，只通过 API 与 TB 交互。

## 监控

- 日志: JSON 格式输出到 stdout
- 状态表: `vendor_price_snapshots` 记录每次抓取结果

## Git 与 GitHub

本地已 `git init`，主线 **`master`**。在 GitHub 上**新建空仓库**（不要自带 README，避免首次 push 冲突），然后：

```bash
git remote add origin https://github.com/<你的用户或组织>/token-bridge-crawler.git
git push -u origin master
```

## License

MIT
