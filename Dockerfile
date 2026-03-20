# =====================================================
# 壹信 IM 后端 Dockerfile
# =====================================================
# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o im-backend .

# =====================================================
# 运行阶段
# =====================================================
FROM alpine:latest

# 安装必要的运行时库
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/im-backend .
COPY --from=builder /app/migrations ./migrations

# 创建非 root 用户
RUN adduser -D -u 1000 appuser
USER appuser

# 暴露端口
# HTTP API 端口
EXPOSE 8080
# WebSocket 端口
EXPOSE 8081

# 启动命令
CMD ["./im-backend"]
