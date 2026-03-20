# 壹信 IM 部署指南

## 环境要求

- Docker >= 20.10
- Docker Compose >= 2.0

## 快速开始

### 方式一：使用 Docker Compose（推荐）

1. **复制环境变量配置文件**

```bash
cp .env.example .env
```

2. **编辑 .env 文件，填入实际配置**

```bash
# 必须修改的配置：
# - DB_PASSWORD (PostgreSQL 密码)
# - JWT_SECRET (JWT 密钥，建议运行: openssl rand -base64 32)

vi .env
```

3. **启动服务**

```bash
# 前台运行（查看日志）
docker-compose up

# 后台运行（推荐）
docker-compose up -d
```

4. **查看服务状态**

```bash
docker-compose ps
docker-compose logs -f im-backend
```

### 方式二：使用已有数据库

如果你已有 PostgreSQL 和 Redis：

1. 只需编辑 `.env` 文件，填入你的数据库地址：

```env
DB_HOST=你的PostgreSQL地址
REDIS_HOST=你的Redis地址
DB_PASSWORD=你的密码
JWT_SECRET=你的密钥
```

2. 构建并运行：

```bash
# 构建镜像
docker build -t im-backend:latest .

# 运行容器
docker run -d \
  --name im-backend \
  -p 8080:8080 \
  -p 8081:8081 \
  --env-file .env \
  im-backend:latest
```

## 部署步骤详解

### 1. 数据库初始化

首次部署需要执行数据库迁移：

```bash
# 进入容器执行迁移
docker-compose exec im-backend /app/im-backend migrate

# 或者手动执行 SQL
docker-compose exec postgres psql -U postgres -d im_db -f /app/migrations/001_initial.sql
docker-compose exec postgres psql -U postgres -d im_db -f /app/migrations/002_add_role_to_users.sql
```

### 2. 验证部署

```bash
# 检查 API 健康状态
curl http://localhost:8080/health

# 检查 WebSocket 连接
wscat -c ws://localhost:8081/ws
```

## 常用命令

```bash
# 重启服务
docker-compose restart im-backend

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down

# 重新构建（代码更新后）
docker-compose build --no-cache im-backend
docker-compose up -d
```

## 端口说明

| 端口 | 用途 |
|------|------|
| 8080 | HTTP API |
| 8081 | WebSocket |
| 5432 | PostgreSQL (可选) |
| 6379 | Redis (可选) |

## 安全建议

1. **JWT_SECRET** - 使用强密钥：`openssl rand -base64 32`
2. **数据库密码** - 使用强密码，不要使用默认密码
3. **生产环境** - 使用外部数据库，不要使用 Docker Compose 自带的
4. **防火墙** - 只开放 8080 和 8081 端口
5. **HTTPS** - 使用 Nginx 或 Traefik 配置 HTTPS 代理
