# 壹信 IM 后端

> 高并发即时通讯系统后端服务

## 环境要求

- **Go**: 1.21+
- **PostgreSQL**: 15+
- **Redis**: 7+

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| Web 框架 | Gin |
| WebSocket | Gorilla WebSocket |
| 数据库 | PostgreSQL 15 |
| 缓存/消息队列 | Redis 7 |
| JWT | golang-jwt/jwt |

## 项目结构

```
backend/
├── api/              # API 路由层
├── config/           # 配置文件
├── middleware/       # 中间件
├── migrations/       # 数据库迁移
├── model/            # 数据模型
├── queue/            # 消息队列
├── repository/       # 数据访问层
├── service/          # 业务逻辑层
├── ws/               # WebSocket
├── main.go           # 入口文件
└── go.mod            # 依赖管理
```

## 快速开始

### 1. 克隆项目

```bash
cd D:\后端工程师
```

### 2. 安装依赖

```bash
cd backend
go mod tidy
```

### 3. 配置环境变量

可以设置以下环境变量，或使用默认值：

```bash
# Server
export SERVER_PORT=8080

# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=im_db

# Redis
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=

# JWT
export JWT_SECRET=your-secret-key
export JWT_EXPIRE=168  # 7天
```

### 4. 初始化数据库

执行迁移脚本：

```bash
psql -U postgres -d im_db -f migrations/001_initial_schema.sql
```

### 5. 启动服务

```bash
go run main.go
```

服务将在 `http://localhost:8080` 启动。

## API 接口

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| POST | /api/auth/refresh | 刷新 Token |

### 用户

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/user/profile | 获取用户资料 |
| PUT | /api/user/profile | 更新用户资料 |
| GET | /api/user/friends | 获取好友列表 |
| POST | /api/user/friend | 添加好友 |

### 会话

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/conversations | 获取会话列表 |
| POST | /api/conversations | 创建会话 |

### 消息

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/messages/send | 发送消息 |
| GET | /api/messages/history | 获取历史消息 |
| POST | /api/messages/revoke | 撤回消息 |
| POST | /api/messages/read | 标记已读 |

### 群组

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/group/create | 创建群组 |
| GET | /api/group/:id | 获取群信息 |
| POST | /api/group/:id/member | 添加成员 |
| DELETE | /api/group/:id/member/:uid | 移除成员 |
| POST | /api/group/:id/mute | 禁言成员 |

### WebSocket

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /ws?token=xxx | WebSocket 连接 |

## WebSocket 消息协议

### 连接

```
Client -> Server: {"type": "connect", "token": "jwt_token"}
Server -> Client: {"type": "connected", "userId": 1}
```

### 心跳

```
Client -> Server: {"type": "ping"}
Server -> Client: {"type": "pong"}
```

### 消息推送

```
Server -> Client: {
  "type": "message",
  "data": {
    "messageId": 1,
    "conversationId": 1,
    "senderId": 2,
    "content": "你好",
    "timestamp": 1700000000
  }
}
```

## 性能特性

- **64 分片锁**: 使用 FNV 哈希降低锁竞争
- **5万缓冲通道**: 应对流量洪峰
- **100+ 消费者**: 并发处理消息
- **Redis Stream**: 高性能消息队列

## 开发规范

### Git 提交

使用约定式提交：

```bash
git commit -m "feat: 添加用户注册功能"
git commit -m "fix: 修复登录 bug"
git commit -m "docs: 更新 API 文档"
```

### 分支命名

```bash
feat/user-auth      # 功能分支
fix/websocket-bug   # 修复分支
docs/api-spec       # 文档分支
```

## 许可证

MIT License
