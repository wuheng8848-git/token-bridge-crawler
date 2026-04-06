# Token Bridge Intelligence - 开发指南

## 环境准备

### 前置依赖

- Go 1.21+
- PostgreSQL 14+
- Git

### 克隆项目

```bash
git clone <repository-url>
cd token-bridge-crawler
```

### 安装依赖

```bash
go mod download
```

## 配置开发环境

### 1. 创建环境变量文件

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```env
# 数据库连接（与 TB 共用或独立）
CRAWLER_DATABASE_URL=postgres://user:password@localhost:5432/tokenbridge?sslmode=disable

# TB API 配置
TB_ADMIN_API_TOKEN=your-admin-api-token
TB_BASE_URL=http://localhost:8080

# AI 日报配置（可选）
CRAWLER_AI_API_KEY=your-ai-api-key

# OpenAI 抓取配置（可选）
OPENAI_API_KEY=your-openai-api-key

# OpenRouter 翻译服务（可选）
OPENROUTER_API_KEY=your-openrouter-api-key

# 邮件配置（可选）
SMTP_HOST=smtp.gmail.com
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
```

### 2. 数据库迁移

```bash
# 使用迁移文件
psql $CRAWLER_DATABASE_URL -f deploy/migrations/001_create_vendor_price_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/002_create_intelligence_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/003_create_marketing_tables.up.sql
```

### 3. 验证配置

```bash
# 测试单次抓取（价格爬虫）
go run ./cmd/crawler -once

# 测试情报系统
go run ./cmd/intelligence -once
```

## 项目结构

```
token-bridge-crawler/
├── cmd/
│   ├── crawler/                    # 价格爬虫入口
│   │   └── main.go
│   └── intelligence/               # 情报系统入口
│       └── main.go
│
├── internal/
│   ├── core/                       # 核心抽象层
│   │   ├── types.go               # IntelType, IntelItem 定义
│   │   ├── collector.go           # Collector 接口
│   │   └── registry.go            # 采集器注册表
│   │
│   ├── collectors/                 # 采集器实现
│   │   ├── price/                 # 价格采集器
│   │   │   ├── base.go
│   │   │   ├── google.go
│   │   │   ├── openai.go
│   │   │   ├── anthropic.go
│   │   │   └── openrouter.go
│   │   │
│   │   ├── apidoc/                # API文档采集器
│   │   │   ├── base.go
│   │   │   └── openai.go
│   │   │
│   │   ├── policy/                # 政策采集器
│   │   │   └── base.go
│   │   │
│   │   ├── userpain/              # 用户痛点采集器
│   │   │   ├── base.go
│   │   │   ├── hackernews.go
│   │   │   ├── reddit.go
│   │   │   └── configpain.go
│   │   │
│   │   ├── tool/                  # 工具生态采集器
│   │   │   ├── base.go
│   │   │   └── ecosystem.go
│   │   │
│   │   ├── integration/           # 集成机会采集器
│   │   │   └── base.go
│   │   │
│   │   ├── useracquisition/       # 用户获取采集器
│   │   │   └── base.go
│   │   │
│   │   ├── conversion/            # 转化情况采集器
│   │   │   └── base.go
│   │   │
│   │   ├── usage/                 # 使用模式采集器
│   │   │   └── base.go
│   │   │
│   │   └── community/             # 社区采集器
│   │       ├── discord.go
│   │       ├── linkedin.go
│   │       └── producthunt.go
│   │
│   ├── storage/                   # 存储层
│   │   ├── storage.go             # 价格存储
│   │   ├── intelligence.go        # 情报存储
│   │   └── translated_storage.go  # 翻译存储包装
│   │
│   ├── marketing/                 # 营销决策层
│   │   ├── types/                 # 类型定义
│   │   │   └── types.go
│   │   ├── detectors/             # 信号检测器
│   │   │   ├── cost_pressure_detector.go
│   │   │   ├── config_friction_detector.go
│   │   │   ├── tool_fragmentation_detector.go
│   │   │   ├── governance_start_detector.go
│   │   │   ├── migration_intent_detector.go
│   │   │   └── general_interest_detector.go
│   │   ├── generators/            # 动作生成器
│   │   │   ├── cost_action_generator.go
│   │   │   ├── config_action_generator.go
│   │   │   ├── tool_action_generator.go
│   │   │   ├── governance_action_generator.go
│   │   │   └── migration_action_generator.go
│   │   └── signal_model.go        # 信号模型
│   │
│   ├── reporter/                  # 报告层
│   │   └── daily.go               # 日报生成器
│   │
│   ├── scheduler/                 # 调度层
│   │   └── intelligence.go        # 情报调度器
│   │
│   ├── ai/                        # AI 服务层
│   │   ├── translator.go          # 翻译服务
│   │   └── translation_service.go # 翻译管理
│   │
│   ├── mail/                      # 邮件服务
│   │   └── sender.go
│   │
│   └── adapters/                  # 厂商适配器（旧）
│       ├── base.go
│       ├── google.go
│       ├── openai.go
│       └── anthropic.go
│
├── deploy/migrations/              # 数据库迁移
│   ├── 001_create_vendor_price_tables.up.sql
│   ├── 001_create_vendor_price_tables.down.sql
│   ├── 002_create_intelligence_tables.up.sql
│   ├── 003_create_marketing_tables.up.sql
│   └── 003_create_marketing_tables.down.sql
│
├── docs/                           # 文档
│   ├── 01-overview.md
│   ├── 02-architecture.md
│   ├── 03-development.md
│   ├── 04-deployment.md
│   ├── 05-api.md
│   ├── 06-intelligence-roadmap.md
│   └── 07-customer-signal-model.md
│
├── config.yaml                     # 配置文件
├── .env.example                    # 环境变量示例
├── Dockerfile                      # 容器镜像
├── README.md                       # 项目说明
└── DEPLOY.md                       # 部署文档
```

## 开发流程

### 添加新采集器

1. 在 `internal/collectors/<type>/` 创建新文件

```go
package price

import (
    "context"
    "time"
    "token-bridge-crawler/internal/core"
)

type NewVendorCollector struct {
    name     string
    source   string
    interval time.Duration
}

func NewNewVendorCollector() *NewVendorCollector {
    return &NewVendorCollector{
        name:     "newvendor_collector",
        source:   "newvendor",
        interval: 24 * time.Hour,
    }
}

func (c *NewVendorCollector) Name() string {
    return c.name
}

func (c *NewVendorCollector) Source() string {
    return c.source
}

func (c *NewVendorCollector) IntelType() core.IntelType {
    return core.IntelTypePrice
}

func (c *NewVendorCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
    // 实现采集逻辑
    return nil, nil
}

func (c *NewVendorCollector) RateLimit() time.Duration {
    return 2 * time.Second
}
```

2. 在 `cmd/intelligence/main.go` 中注册采集器

```go
registry.Register(newVendorCollector)
log.Println("[Registry] 注册 NewVendor 采集器")
```

### 添加新信号检测器

1. 在 `internal/marketing/detectors/` 创建新文件

```go
package detectors

import (
    "token-bridge-crawler/internal/core"
    "token-bridge-crawler/internal/marketing/types"
)

type NewSignalDetector struct{}

func NewNewSignalDetector() *NewSignalDetector {
    return &NewSignalDetector{}
}

func (d *NewSignalDetector) GetSupportedTypes() []types.SignalType {
    return []types.SignalType{types.SignalTypeNewSignal}
}

func (d *NewSignalDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
    // 实现检测逻辑
    return nil, nil
}
```

2. 在 `internal/marketing/signal_model.go` 中注册检测器

### 调试单个采集器

```bash
# 运行情报系统单次模式
go run ./cmd/intelligence -once
```

### 本地运行定时任务

```bash
# 启动情报系统定时服务
go run ./cmd/intelligence
```

## 测试

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/storage/...

# 带覆盖率
go test -cover ./...
```

### 编译验证

```bash
# 编译所有包
go build ./...

# 编译情报系统
go build -o intelligence.exe ./cmd/intelligence
```

## 常见问题

### 1. 数据库连接失败

```
fatal: database "tokenbridge" does not exist
```

**解决**: 创建数据库
```bash
createdb tokenbridge
```

### 2. 表不存在

```
ERROR: relation "intelligence_items" does not exist
```

**解决**: 执行迁移
```bash
psql $CRAWLER_DATABASE_URL -f deploy/migrations/002_create_intelligence_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/003_create_marketing_tables.up.sql
```

### 3. 编译错误

**排查步骤**:
1. 检查 Go 版本是否 >= 1.21
2. 运行 `go mod tidy`
3. 检查导入路径是否正确

## 代码规范

### 命名规范

- **文件**: 小写 + 下划线，如 `cost_pressure_detector.go`
- **结构体**: 大驼峰，如 `CostPressureDetector`
- **接口**: 大驼峰 + `er` 后缀，如 `Detector`
- **函数**: 大驼峰，如 `DetectFromIntel`

### 错误处理

```go
// 包装错误，添加上下文
if err != nil {
    return fmt.Errorf("detect signal failed: %w", err)
}
```

### 日志规范

```go
// 使用标准库 log
log.Printf("[Detector] 检测到 %d 个信号", len(signals))
log.Printf("[Generator] 生成 %d 个动作", len(actions))
```

## 提交规范

```
feat: 添加 NewVendor 采集器
fix: 修复信号检测器误判问题
docs: 更新架构文档
refactor: 重构存储层逻辑
test: 添加单元测试
```