# Token Bridge Intelligence - 部署文档

## 概述

Token Bridge Intelligence 是情报感知系统，用于采集市场情报、检测客户信号、生成内部建议。

**重要**：本项目不是自动化营销系统，所有动作停留在内部建议层，不自动外发。

## 部署方式概览

| 方式 | 适用场景 | 复杂度 |
|------|----------|--------|
| Docker Compose | 本地开发/测试 | 低 |
| Kubernetes CronJob | 生产环境 | 中 |
| 直接部署 | 简单环境 | 低 |

## 端口与健康检查口径

- `TB_BASE_URL` 指向 TBv2 的后端 API（常见为 `8080`），用于调用 Admin API（如 `POST /v1/admin/supplier_catalog_staging/import`）
- 本项目自身会启动一个 HTTP API/健康检查服务，端口由 `API_PORT` 控制，默认 `8081`，用于避免与 TBv2 的 `8080` 抢占端口
- 健康检查路径：`GET /healthz`（示例：`curl http://127.0.0.1:8081/healthz`）

## 方式一：Docker Compose（推荐开发/测试）

### 1. 构建镜像

```bash
docker build -t token-bridge-intelligence:latest .
```

### 2. 在 TB 项目的 docker-compose.yml 中添加

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
      - TB_BASE_URL=http://api:8080
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

### 3. 启动服务

```bash
docker-compose up -d intelligence
```

## 方式二：Kubernetes CronJob（推荐生产）

### 1. 创建 Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: intelligence-secrets
  namespace: token-bridge
type: Opaque
stringData:
  database-url: "postgres://user:pass@postgres:5432/tokenbridge"
  tb-token: "your-admin-api-token"
  ai-api-key: "your-ai-api-key"
  openrouter-api-key: "your-openrouter-api-key"
  smtp-password: "your-smtp-password"
```

### 2. 创建 ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: intelligence-config
  namespace: token-bridge
data:
  config.yaml: |
    scheduler:
      cron: "0 2 * * *"
    collectors:
      price:
        google:
          enabled: true
        openai:
          enabled: true
        anthropic:
          enabled: true
        openrouter:
          enabled: true
      apidoc:
        openai:
          enabled: true
      userpain:
        hackernews:
          enabled: true
        reddit:
          enabled: true
      tool:
        ecosystem:
          enabled: true
    tb_api:
      base_url: "http://tb-api:8080"
      batch_size: 50
    ai_report:
      enabled: true
      provider:
        name: "openrouter"
        model: "deepseek-chat"
    email:
      enabled: true
      smtp:
        host: "smtp.gmail.com"
        port: 587
        tls: true
      from: "intelligence@tokenbridge.local"
      to:
        - "ops@tokenbridge.local"
```

### 3. 创建 CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: token-bridge-intelligence
  namespace: token-bridge
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: intelligence
            image: your-registry/token-bridge-intelligence:latest
            imagePullPolicy: Always
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
            - name: CRAWLER_AI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: intelligence-secrets
                  key: ai-api-key
            - name: OPENROUTER_API_KEY
              valueFrom:
                secretKeyRef:
                  name: intelligence-secrets
                  key: openrouter-api-key
            - name: SMTP_PASS
              valueFrom:
                secretKeyRef:
                  name: intelligence-secrets
                  key: smtp-password
            volumeMounts:
            - name: config
              mountPath: /root/config.yaml
              subPath: config.yaml
            resources:
              requests:
                memory: "128Mi"
                cpu: "100m"
              limits:
                memory: "256Mi"
                cpu: "200m"
          volumes:
          - name: config
            configMap:
              name: intelligence-config
          restartPolicy: OnFailure
```

### 4. 部署

```bash
kubectl apply -f intelligence-secret.yaml
kubectl apply -f intelligence-configmap.yaml
kubectl apply -f intelligence-cronjob.yaml
```

### 5. 手动触发测试

```bash
kubectl create job --from=cronjob/token-bridge-intelligence intelligence-test -n token-bridge
kubectl logs -f job/intelligence-test -n token-bridge
```

## 方式三：直接部署

### 1. 准备服务器

```bash
# 创建目录
mkdir -p /opt/token-bridge-intelligence
cd /opt/token-bridge-intelligence

# 上传二进制文件或源码
git clone <repo> .
```

### 2. 构建

```bash
go build -o intelligence ./cmd/intelligence
```

### 3. 配置

```bash
cp .env.example .env
# 编辑 .env 填入配置
```

### 4. 使用 systemd 管理

创建 `/etc/systemd/system/token-bridge-intelligence.service`:

```ini
[Unit]
Description=Token Bridge Intelligence
After=network.target

[Service]
Type=simple
User=intelligence
WorkingDirectory=/opt/token-bridge-intelligence
ExecStart=/opt/token-bridge-intelligence/intelligence
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable token-bridge-intelligence
sudo systemctl start token-bridge-intelligence
sudo systemctl status token-bridge-intelligence
```

### 5. 或使用 Crontab

```bash
# 编辑 crontab
crontab -e

# 添加定时任务（每天凌晨 2 点）
0 2 * * * cd /opt/token-bridge-intelligence && ./intelligence -once >> /var/log/intelligence.log 2>&1
```

## 环境变量配置

| 变量名 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `CRAWLER_DATABASE_URL` | 是 | 数据库连接 | `postgres://user:pass@localhost:5432/tb` |
| `TB_ADMIN_API_TOKEN` | 是 | TB Admin API Token | `sk-xxx` |
| `TB_BASE_URL` | 否 | TB API 地址 | `http://localhost:8080` |
| `CRAWLER_AI_API_KEY` | 否 | AI 日报 API Key | `sk-xxx` |
| `OPENROUTER_API_KEY` | 否 | OpenRouter 翻译服务 | `sk-xxx` |
| `OPENAI_API_KEY` | 否 | OpenAI 抓取用 | `sk-xxx` |
| `SMTP_HOST` | 否 | SMTP 服务器 | `smtp.gmail.com` |
| `SMTP_USER` | 否 | SMTP 用户名 | `user@gmail.com` |
| `SMTP_PASS` | 否 | SMTP 密码 | `app-password` |

## 数据库迁移

```bash
# 执行所有迁移
psql $CRAWLER_DATABASE_URL -f deploy/migrations/001_create_vendor_price_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/002_create_intelligence_tables.up.sql
psql $CRAWLER_DATABASE_URL -f deploy/migrations/003_create_marketing_tables.up.sql
```

## 监控与告警

### 日志检查

```bash
# Docker
docker logs tbv2-intelligence

# Kubernetes
kubectl logs -n token-bridge cronjob/token-bridge-intelligence

# 直接部署
journalctl -u token-bridge-intelligence -f
```

### 数据库查询

```sql
-- 查看最近情报采集记录
SELECT intel_type, source, status, items_count, started_at
FROM collector_runs
ORDER BY started_at DESC
LIMIT 10;

-- 查看最近检测到的信号
SELECT signal_type, strength, platform, status, detected_at
FROM customer_signals
ORDER BY detected_at DESC
LIMIT 10;

-- 查看生成的内部建议
SELECT action_type, channel, status, priority, created_at
FROM marketing_actions
ORDER BY created_at DESC
LIMIT 10;

-- 确认动作未自动执行
SELECT COUNT(*) as draft_count
FROM marketing_actions
WHERE status = 'draft';
```

## 故障排查

### 问题：情报采集返回空数据

**排查步骤**:
1. 检查数据源是否可访问
2. 检查是否被限流
3. 手动测试：

```bash
docker run --rm token-bridge-intelligence ./intelligence -once
```

### 问题：邮件发送失败

**排查步骤**:
1. 检查 SMTP 配置
2. 检查防火墙是否允许 587 端口
3. 查看日志中的具体错误

### 问题：数据未导入 TB

**排查步骤**:
1. 检查 `TB_ADMIN_API_TOKEN` 是否有效
2. 检查 TB API 是否可访问
3. 查看 TB API 日志

## 升级维护

### 更新镜像

```bash
# 拉取新镜像
docker pull your-registry/token-bridge-intelligence:latest

# 重启服务
docker-compose up -d intelligence

# 或 Kubernetes
kubectl rollout restart cronjob/token-bridge-intelligence -n token-bridge
```

### 数据库迁移

```bash
# 执行新迁移
psql $CRAWLER_DATABASE_URL -f deploy/migrations/003_create_marketing_tables.up.sql
```

## 备份策略

```bash
# 备份情报数据
pg_dump $CRAWLER_DATABASE_URL \
  --table=intelligence_items \
  --table=customer_signals \
  --table=marketing_actions \
  --table=collector_runs \
  > intelligence_backup_$(date +%Y%m%d).sql
```

---

**文档维护**：Token Bridge Team
**最后更新**：2026-03-31
**版本**：v2.0
