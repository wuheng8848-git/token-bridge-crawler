# Token Bridge Intelligence 容器化部署指南

## 概述

本指南适用于在便宜云服务器（2核2G）上部署 Token Bridge Intelligence 情报系统。

**部署架构**:
```
云服务器
├── Docker Compose
│   ├── intelligence (情报系统 + API)
│   └── postgres (PostgreSQL 14)
```

---

## 1. 准备云服务器

### 推荐配置
- **CPU**: 2核
- **内存**: 2GB
- **磁盘**: 50GB SSD
- **带宽**: 3Mbps（足够）
- **系统**: Ubuntu 22.04 LTS

### 推荐厂商
| 厂商 | 配置 | 价格 |
|------|------|------|
| 阿里云轻量 | 2核2G 50GB | ~60元/月 |
| 腾讯云轻量 | 2核2G 50GB | ~60元/月 |
| 华为云HECS | 2核2G | ~50元/月 |

---

## 2. 安装 Docker

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装 Docker
sudo apt install -y docker.io docker-compose

# 启动 Docker
sudo systemctl enable docker
sudo systemctl start docker

# 验证安装
docker --version
docker-compose --version
```

---

## 3. 部署步骤

### 3.1 上传代码

```bash
# 在云服务器上创建目录
mkdir -p /opt/token-bridge-intelligence
cd /opt/token-bridge-intelligence

# 方式1: 从本地复制（在本地执行）
scp -r /path/to/token-bridge-crawler root@your-server-ip:/opt/

# 方式2: 从 Git 克隆
git clone <your-repo-url> .
```

### 3.2 配置环境变量

```bash
cd /opt/token-bridge-crawler/deploy
cp .env.example .env

# 编辑 .env 文件
nano .env
```

**最小化配置**（只填数据库密码）:
```env
# 数据库配置（必填）
DB_USER=tbuser
DB_PASSWORD=your_secure_password_here
DB_NAME=tokenbridge

# Tavily API Key（推荐配置，用于搜索引擎采集）
# 获取地址: https://tavily.com (免费版每月1000次搜索)
TAVILY_API_KEY=tvly-dev-xxxxxxxxxxxx

# 其他配置可以先不填，系统会跳过相关功能
```

### 3.3 启动服务

```bash
# 进入部署目录
cd /opt/token-bridge-crawler/deploy

# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f intelligence
```

首次启动会自动:
1. 拉取 PostgreSQL 14 镜像
2. 构建情报系统镜像
3. 执行数据库迁移
4. 启动情报系统服务

---

## 4. 验证部署

### 4.1 检查服务状态

```bash
# 查看容器状态
docker-compose ps

# 预期输出:
# NAME                   STATUS
# tb-intelligence        Up 2 minutes
# tb-intelligence-db     Up 2 minutes
```

### 4.2 测试 API

```bash
# 健康检查
curl http://localhost:8081/healthz

# 预期输出:
# {"status":"ok","timestamp":"2026-04-03T..."}

# 查看采集器列表
curl http://localhost:8081/api/v1/collectors

# 查看情报统计
curl http://localhost:8081/api/v1/stats/intelligence

# 查看信号统计
curl http://localhost:8081/api/v1/stats/signals

# 查看采集器运行记录
curl http://localhost:8081/api/v1/collector-runs
```

### 4.3 查看日志

```bash
# 实时查看情报系统日志
docker-compose logs -f intelligence

# 查看最近100行
docker-compose logs --tail=100 intelligence
```

---

## 5. 配置定时采集

编辑 `config.yaml` 配置采集频率:

```bash
nano /opt/token-bridge-crawler/deploy/config.yaml
```

**示例配置**:
```yaml
scheduler:
  # 默认每小时检查一次
  cron: "0 * * * *"

collectors:
  price:
    google:
      enabled: true
      cron: "0 2 * * *"      # 每天凌晨2点采集
    openai:
      enabled: true
      cron: "0 3 * * *"      # 每天凌晨3点采集
    anthropic:
      enabled: true
      cron: "0 4 * * *"
    openrouter:
      enabled: true
      cron: "0 */6 * * *"     # 每6小时采集

  userpain:
    hackernews:
      enabled: true
      cron: "0 */2 * * *"     # 每2小时采集
    reddit:
      enabled: true
      cron: "0 */3 * * *"

  # 其他采集器...
```

修改配置后重启:
```bash
docker-compose restart intelligence
```

---

## 6. 日常运维

### 6.1 查看运行状态

```bash
# 进入部署目录
cd /opt/token-bridge-crawler/deploy

# 查看容器状态
docker-compose ps

# 查看资源使用
docker stats
```

### 6.2 手动触发采集

```bash
# 进入容器执行单次采集
docker-compose exec intelligence ./intelligence -once
```

### 6.3 备份数据库

```bash
# 备份到文件
docker-compose exec postgres pg_dump -U tbuser tokenbridge > backup_$(date +%Y%m%d).sql

# 自动备份脚本（添加到 crontab）
# 0 3 * * * cd /opt/token-bridge-crawler/deploy && docker-compose exec -T postgres pg_dump -U tbuser tokenbridge > /backup/tb_$(date +\%Y\%m\%d).sql
```

### 6.4 更新部署

```bash
# 拉取最新代码
git pull

# 重建镜像
docker-compose build --no-cache

# 重启服务
docker-compose up -d
```

### 6.5 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎使用）
docker-compose down -v
```

---

## 7. 故障排查

### 问题1: 容器启动失败

```bash
# 查看详细日志
docker-compose logs intelligence

# 检查环境变量
cat .env

# 检查配置文件
cat config.yaml
```

### 问题2: 数据库连接失败

```bash
# 检查数据库容器
docker-compose logs postgres

# 测试数据库连接
docker-compose exec postgres psql -U tbuser -d tokenbridge -c "SELECT 1"
```

### 问题3: API 无法访问

```bash
# 检查端口监听
netstat -tlnp | grep 8081

# 检查防火墙
sudo ufw status
# 或
sudo iptables -L -n | grep 8081
```

### 问题4: 内存不足

```bash
# 查看内存使用
free -h

# 添加 Swap（如果内存不足）
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

---

## 8. 安全建议

### 8.1 防火墙配置

```bash
# 只开放必要的端口
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 8081/tcp    # 情报系统 API
sudo ufw enable
```

### 8.2 数据库安全

- PostgreSQL 只监听本地（已通过 docker-compose 配置）
- 使用强密码
- 定期备份

### 8.3 API 安全

当前 API 没有认证，建议:
- 使用 Nginx 反向代理添加 Basic Auth
- 或限制只允许本地/内网访问

---

## 9. 性能优化

### 9.1 数据库优化

```bash
# 进入数据库
docker-compose exec postgres psql -U tbuser -d tokenbridge

# 添加索引（如果查询慢）
CREATE INDEX idx_intelligence_items_type ON intelligence_items(intel_type);
CREATE INDEX idx_intelligence_items_time ON intelligence_items(captured_at);
CREATE INDEX idx_collector_runs_time ON collector_runs(started_at);
```

### 9.2 日志清理

```bash
# 清理旧日志
docker-compose exec intelligence find /root/logs -name "*.log" -mtime +7 -delete
```

---

## 10. 常用命令速查

```bash
# 启动
docker-compose up -d

# 停止
docker-compose down

# 重启
docker-compose restart

# 查看日志
docker-compose logs -f

# 进入容器
docker-compose exec intelligence sh

# 数据库操作
docker-compose exec postgres psql -U tbuser -d tokenbridge

# 查看统计
curl http://localhost:8081/api/v1/stats/intelligence | jq
```

---

## 11. 更新日志

### v1.1 (2026-04-06)
- **新增**: Tavily 搜索引擎采集器支持
  - 需要配置 `TAVILY_API_KEY`
  - 支持迁移意愿、成本压力等信号采集
- **修复**: API 端口从 8080 改为 8081（避免与主项目冲突）
- **修复**: 健康检查端点 `/health` 改为 `/healthz`
- **优化**: 规则引擎支持数据库持久化（21条规则）
- **优化**: 翻译策略采用保守模式（质量分≥60才翻译）

### v1.0 (2026-04-03)
- 初始版本
- 支持价格采集、用户痛点采集
- 基础 API 服务

---

**文档维护**: Token Bridge Team
**最后更新**: 2026-04-06
**版本**: v1.1
