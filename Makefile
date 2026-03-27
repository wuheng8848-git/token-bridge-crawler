# Token Bridge Crawler Makefile

.PHONY: build run test clean docker migrate help

# 变量
BINARY_NAME=crawler
DOCKER_IMAGE=token-bridge-crawler:latest
MAIN_FILE=cmd/crawler/main.go

# 默认目标
help:
	@echo "Token Bridge Crawler - 可用命令:"
	@echo "  make build       - 构建二进制文件"
	@echo "  make run         - 运行服务"
	@echo "  make run-once    - 单次执行（测试）"
	@echo "  make test        - 运行测试"
	@echo "  make clean       - 清理构建文件"
	@echo "  make docker      - 构建 Docker 镜像"
	@echo "  make migrate-up  - 执行数据库迁移"
	@echo "  make deps        - 安装依赖"
	@echo "  make fmt         - 格式化代码"
	@echo "  make lint        - 代码检查"

# 构建
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_FILE)

# 运行服务
run:
	@echo "Starting crawler service..."
	go run $(MAIN_FILE)

# 单次执行（测试）
run-once:
	@echo "Running once..."
	go run $(MAIN_FILE) -once

# 测试
test:
	@echo "Running tests..."
	go test -v ./...

# 清理
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf vendor/

# Docker 构建
docker:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# 数据库迁移（需要 DATABASE_URL 环境变量）
migrate-up:
	@echo "Running migrations..."
	psql $(DATABASE_URL) -f deploy/migrations/001_create_vendor_price_tables.up.sql

migrate-down:
	@echo "Rolling back migrations..."
	psql $(DATABASE_URL) -f deploy/migrations/001_create_vendor_price_tables.down.sql

# 依赖管理
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# 代码格式化
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# 开发模式（带热重载）
dev:
	@echo "Running in dev mode..."
	air -c .air.toml

# 抓取特定厂商（调试用）
crawl-google:
	@echo "Crawling Google..."
	go run $(MAIN_FILE) -once -vendor=google

crawl-openai:
	@echo "Crawling OpenAI..."
	go run $(MAIN_FILE) -once -vendor=openai

crawl-anthropic:
	@echo "Crawling Anthropic..."
	go run $(MAIN_FILE) -once -vendor=anthropic
