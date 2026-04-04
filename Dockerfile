# Token Bridge Intelligence Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建 intelligence 入口
RUN CGO_ENABLED=0 GOOS=linux go build -o intelligence ./cmd/intelligence

# 运行镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从 builder 复制二进制文件
COPY --from=builder /app/intelligence .

# 复制配置文件（也可以通过挂载覆盖）
COPY config.yaml .

# 暴露 API 端口
EXPOSE 8080

# 默认运行 intelligence 服务
CMD ["./intelligence"]
