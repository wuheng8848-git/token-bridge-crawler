# 速查表

> 快速参考，详细说明见 [index.md](../index.md)

---

## 常用命令

```bash
# 单次执行（测试）
go run ./cmd/intelligence -once

# 启动定时服务
go run ./cmd/intelligence

# 编译
go build -o intelligence ./cmd/intelligence

# 运行测试
go test ./...

# 数据库迁移
psql $CRAWLER_DATABASE_URL -f deploy/migrations/002_create_intelligence_tables.up.sql
```

---

## 环境变量

```bash
# 数据库
CRAWLER_DATABASE_URL=postgres://tbv2:tbv2_password@localhost:15432/token_bridge_crawler?sslmode=disable

# 翻译服务
BAIDU_APP_ID=xxx
BAIDU_API_KEY=xxx              # 大模型翻译
BAIDU_APP_SECRET=xxx           # 通用翻译
VOLCENGINE_ACCESS_KEY_ID=xxx
VOLCENGINE_SECRET_ACCESS_KEY=xxx

# 主项目推送（可选）
TB_BASE_URL=http://localhost:3000
TB_ADMIN_API_TOKEN=xxx
```

---

## 数据表

| 表名 | 用途 |
|------|------|
| `intelligence_items` | 情报主表 |
| `customer_signals` | 客户信号 |
| `marketing_actions` | 营销动作（内部建议） |
| `vendor_price_snapshots` | 价格快照 |
| `vendor_price_details` | 价格明细 |

---

## 常用查询

```sql
-- 情报统计
SELECT COUNT(*) FROM intelligence_items;

-- 翻译覆盖率
SELECT COUNT(*) FILTER (WHERE metadata ? 'title_zh') FROM intelligence_items;

-- 未处理信号
SELECT * FROM customer_signals WHERE status = 'new' LIMIT 10;

-- 待审核动作
SELECT * FROM marketing_actions WHERE status = 'draft' LIMIT 10;
```

---

## 翻译器优先级

```
火山引擎（200万/月，172ms）→ 百度大模型（100万/月）→ 百度通用（5万/月）
```

---

## 信号类型

| 信号 | 优先级 |
|------|--------|
| `cost_pressure` | P0 |
| `config_friction` | P0 |
| `tool_fragmentation` | P0 |
| `governance_start` | P1 |
| `migration_intent` | P1 |
| `general_interest` | P3 |

---

## 入口文件

| 入口 | 说明 |
|------|------|
| `cmd/intelligence/main.go` | 主干入口 ⭐ |
| `cmd/crawler/main.go` | 旧爬虫（维护中） |