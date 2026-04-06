# Token Bridge Intelligence 部署指南

## 概述

Token Bridge Intelligence 是情报感知系统，用于采集市场情报、检测客户信号、生成内部建议。

**重要**：本项目不是自动化营销系统，所有动作停留在内部建议层，不自动外发。

## 当前阶段判断

### 已完成

- ✅ 多类型情报采集器
- ✅ 统一情报存储
- ✅ 信号检测器
- ✅ 动作生成器
- ✅ 信号/动作持久化

### 已接上的链路

```
ProcessIntel(...) → SaveSignals(...) → SaveActions(...)
```

### 当前动作层边界

| 属性 | 值 | 说明 |
|------|-----|------|
| Channel | `internal` | 统一使用内部渠道 |
| AutoExecute | `false` | 不自动执行 |
| Status | `draft` | 草稿状态，需人工审核 |

## 部署方式

### 方式一：Docker Compose（推荐）

在 TB 项目的 `deploy/local/docker-compose.yml` 中添加：

```yaml
services:
  intelligence:
    build:
      context: ../../token-bridge-crawler
      dockerfile: Dockerfile
    container_name: tbv2-intelligence
    environment:
      - CRAWLER_DATABASE_URL=${DATABASE_URL}
      - TB_ADMIN_API_TOKEN=${TB_ADMIN_API_TOKEN}
      - CRAWLER_AI_API_KEY=${CRAWLER_AI_API_KEY}
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_USER=${SMTP_USER}
      - SMTP_PASS=${SMTP_PASS}
    volumes:
      - ./intelligence-config.yaml:/root/config.yaml:ro
    depends_on:
      - api
      - postgres
    restart: unless-stopped
    command: ["./intelligence"]
```

### 方式二：Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: token-bridge-intelligence
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: intelligence
            image: token-bridge-intelligence:latest
            env:
            - name: CRAWLER_DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: intelligence-secrets
                  key: database-url
            - name: TB_ADMIN_API_TOKEN
              valueFrom:
                secretKeyRef:
                  name: intelligence-secrets
                  key: tb-token
            resources:
              requests:
                memory: "128Mi"
                cpu: "100m"
              limits:
                memory: "256Mi"
                cpu: "200m"
          restartPolicy: OnFailure
```

### 方式三：直接部署

```bash
# 1. 克隆代码
git clone <repo> /opt/token-bridge-intelligence
cd /opt/token-bridge-intelligence

# 2. 构建（使用情报系统入口）
go build -o intelligence ./cmd/intelligence

# 3. 配置环境变量
cp .env.example .env
# 编辑 .env

# 4. 运行迁移
psql $CRAWLER_DATABASE_URL -f deploy/migrations/001_create_vendor_price_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/002_create_intelligence_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/003_create_marketing_tables.up.sql

# 5. 测试单次运行
./intelligence -once

# 6. 配置 systemd 服务或 crontab
# Crontab 示例：
# 0 2 * * * cd /opt/token-bridge-intelligence && ./intelligence -once >> /var/log/intelligence.log 2>&1
```

## 环境变量配置

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `CRAWLER_DATABASE_URL` | 数据库连接字符串 | `postgres://user:pass@localhost:5432/tb` |
| `TB_ADMIN_API_TOKEN` | TB Admin API Token | `sk-xxx` |
| `TB_BASE_URL` | TB API 地址 | `http://localhost:8080` |
| `CRAWLER_AI_API_KEY` | AI 日报用的 API Key | `sk-xxx` |
| `OPENROUTER_API_KEY` | OpenRouter 翻译服务 | `sk-xxx` |
| `OPENAI_API_KEY` | OpenAI API Key（用于抓取） | `sk-xxx` |
| `SMTP_HOST` | SMTP 服务器 | `smtp.gmail.com` |
| `SMTP_USER` | SMTP 用户名 | `user@gmail.com` |
| `SMTP_PASS` | SMTP 密码 | `app-password` |

## 数据库迁移

情报系统需要以下表：

```bash
# 价格相关表（旧）
psql $DATABASE_URL -f deploy/migrations/001_create_vendor_price_tables.up.sql

# 情报相关表
psql $DATABASE_URL -f deploy/migrations/002_create_intelligence_tables.up.sql

# 营销相关表
psql $DATABASE_URL -f deploy/migrations/003_create_marketing_tables.up.sql
```

## 监控与告警

### 日志检查

```bash
# 查看最近情报采集记录
psql $DATABASE_URL -c "SELECT intel_type, source, status, items_count FROM collector_runs ORDER BY started_at DESC LIMIT 10;"

# 查看检测到的信号
psql $DATABASE_URL -c "SELECT signal_type, strength, platform FROM customer_signals ORDER BY detected_at DESC LIMIT 10;"

# 确认动作未自动执行
psql $DATABASE_URL -c "SELECT status, COUNT(*) FROM marketing_actions GROUP BY status;"
```

## 故障排查

### 问题：情报采集返回空数据

1. 检查数据源是否可访问
2. 检查是否被限流
3. 手动测试：

```bash
go run ./cmd/intelligence -once
```

### 问题：邮件发送失败

1. 检查 SMTP 配置
2. 检查防火墙是否允许 587 端口
3. 查看日志中的具体错误

### 问题：数据未导入 TB

1. 检查 `TB_ADMIN_API_TOKEN` 是否有效
2. 检查 TB API 是否可访问
3. 查看 TB API 日志

## 升级维护

### 添加新采集器

1. 在 `internal/collectors/<type>/` 创建新的采集器
2. 在 `config.yaml` 中添加配置
3. 在 `cmd/intelligence/main.go` 中注册采集器

### 添加新信号检测器

1. 在 `internal/marketing/detectors/` 创建新的检测器
2. 在 `internal/marketing/signal_model.go` 中注册检测器

---

**文档维护**：Token Bridge Team
**最后更新**：2026-03-31
**版本**：v2.0