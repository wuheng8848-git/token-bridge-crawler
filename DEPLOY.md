# Token Bridge Crawler 部署指南

## 概述

Token Bridge Crawler 是一个独立的爬虫服务，用于抓取 Google、OpenAI、Anthropic 等厂商的模型刊例价，并导入到 TB 系统中。

## 部署方式

### 方式一：Docker Compose（推荐）

在 TB 项目的 `deploy/local/docker-compose.yml` 中添加：

```yaml
services:
  crawler:
    build:
      context: ../../token-bridge-crawler
      dockerfile: Dockerfile
    container_name: tbv2-crawler
    environment:
      - CRAWLER_DATABASE_URL=${DATABASE_URL}
      - TB_ADMIN_API_TOKEN=${TB_ADMIN_API_TOKEN}
      - CRAWLER_AI_API_KEY=${CRAWLER_AI_API_KEY}
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_USER=${SMTP_USER}
      - SMTP_PASS=${SMTP_PASS}
    volumes:
      - ./crawler-config.yaml:/root/config.yaml:ro
    depends_on:
      - api
      - postgres
    restart: unless-stopped
    # 使用 cron 模式运行
    command: ["./crawler"]
```

### 方式二：Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: token-bridge-crawler
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: crawler
            image: token-bridge-crawler:latest
            env:
            - name: CRAWLER_DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: crawler-secrets
                  key: database-url
            - name: TB_ADMIN_API_TOKEN
              valueFrom:
                secretKeyRef:
                  name: crawler-secrets
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
git clone <repo> /opt/token-bridge-crawler
cd /opt/token-bridge-crawler

# 2. 构建
go build -o crawler ./cmd/crawler

# 3. 配置环境变量
cp .env.example .env
# 编辑 .env

# 4. 运行迁移
make migrate-up

# 5. 测试单次运行
./crawler -once

# 6. 配置 systemd 服务或 crontab
# Crontab 示例：
# 0 2 * * * cd /opt/token-bridge-crawler && ./crawler -once >> /var/log/crawler.log 2>&1
```

## 环境变量配置

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `CRAWLER_DATABASE_URL` | 数据库连接字符串 | `postgres://user:pass@localhost:5432/tb` |
| `TB_ADMIN_API_TOKEN` | TB Admin API Token | `sk-xxx` |
| `TB_BASE_URL` | TB API 地址 | `http://localhost:8080` |
| `CRAWLER_AI_API_KEY` | AI 日报用的 API Key | `sk-xxx` |
| `OPENAI_API_KEY` | OpenAI API Key（用于抓取） | `sk-xxx` |
| `SMTP_HOST` | SMTP 服务器 | `smtp.gmail.com` |
| `SMTP_USER` | SMTP 用户名 | `user@gmail.com` |
| `SMTP_PASS` | SMTP 密码 | `app-password` |

## 数据库迁移

爬虫需要以下表：

```bash
# 在 TB 数据库中执行
psql $DATABASE_URL -f migrations/000016_vendor_price_history.up.sql
```

或在 TB 项目中执行：

```bash
cd token-bridge-v2/apps/api
go run ./cmd/migrate -command=up
```

## 监控与告警

### 日志检查

```bash
# 查看最近抓取记录
psql $DATABASE_URL -c "SELECT vendor, snapshot_date, total_models, status FROM vendor_price_snapshots ORDER BY snapshot_date DESC LIMIT 10;"

# 查看失败记录
psql $DATABASE_URL -c "SELECT * FROM vendor_price_snapshots WHERE status != 'success' ORDER BY snapshot_date DESC;"
```

### Prometheus 指标（可选）

可添加以下指标暴露：

```go
// crawler_last_success_timestamp{vendor}
// crawler_models_total{vendor}
// crawler_models_changed{vendor, type="new|updated|removed"}
```

## 故障排查

### 问题：抓取返回空数据

1. 检查厂商页面结构是否变化
2. 检查是否被限流（查看 `vendor_price_snapshots.error_log`）
3. 手动测试适配器：

```bash
go run ./cmd/crawler -once -vendor=google
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

### 更新价格表

当 OpenAI/Google 发布新模型时：

1. 更新 `internal/adapters/openai.go` 中的 `getPricingTable()`
2. 重新构建镜像
3. 部署

### 添加新厂商

1. 在 `internal/adapters/` 创建新的适配器
2. 在 `config.yaml` 中添加配置
3. 更新 `cmd/crawler/main.go` 中的适配器初始化

## 附录：手动测试脚本

```powershell
# Windows PowerShell
.\scripts\test-crawler.ps1 -Vendor google -DatabaseURL "postgres://..."
```

```bash
# Linux/Mac
./scripts/test-crawler.sh google "postgres://..."
```
