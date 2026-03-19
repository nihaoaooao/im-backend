# 壹信 IM 后端架构设计

## 一、技术栈选型

| 层级 | 技术选型 | 理由 |
|------|----------|------|
| **语言/框架** | Go + Gin + Gorilla WebSocket | 高并发、成熟稳定 |
| **数据库** | PostgreSQL + Redis | 关系型存储 + 缓存/消息队列 |
| **消息队列** | Redis Stream | 轻量级、支持消费组 |
| **服务注册** | Consul / Nacos | 服务发现与健康检查 |
| **负载均衡** | Nginx | WebSocket 负载均衡 |

---

## 二、系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        客户端 (Web/iOS/Android)                  │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Nginx (LB + WSS 终止)                        │
│                     端口: 80/443                                 │
└─────────────────────────────────────────────────────────────────┘
                                 │
           ┌──────────────────────┼──────────────────────┐
           ▼                      ▼                      ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  WebSocket 网关1 │  │  WebSocket 网关2 │  │  WebSocket 网关N │
│   :8081          │  │   :8082          │  │   :8083          │
└──────────────────┘  └──────────────────┘  └──────────────────┘
           │                      │                      │
           └──────────────────────┼──────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API 网关 (Gin)                              │
│              统一入口 + 认证 + 限流 + 路由                        │
└─────────────────────────────────────────────────────────────────┘
                                  │
        ┌─────────────────────────┼─────────────────────────┐
        ▼                         ▼                         ▼
┌───────────────┐        ┌───────────────┐        ┌───────────────┐
│  用户服务     │        │  消息服务     │        │  群组服务     │
│  /api/user    │        │  /api/msg     │        │  /api/group   │
└───────────────┘        └───────────────┘        └───────────────┘
        │                         │                         │
        └─────────────────────────┼─────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Redis Cluster                                │
│         ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│         │Session   │  │Message   │  │Cache     │              │
│         │Queue     │  │Queue     │  │          │              │
│         └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                  PostgreSQL 主库 + 读副本                       │
│     users | messages | conversations | groups | group_members │
└─────────────────────────────────────────────────────────────────┘
```

---

## 三、核心模块设计

### 1. WebSocket 网关服务

**职责**：
- 维护长连接
- 消息推送
- 心跳检测
- 连接认证

**关键特性**：
- FNV Hash 分片锁（64分片）
- 多 Worker 并行（连接/广播分离）
- 5w+ 缓冲通道应对流量洪峰

### 2. 消息服务

**职责**：
- 消息收发
- 消息存储（分表）
- ACK 确认
- 已读回执
- 消息撤回

**消息分表策略**：
```sql
-- 按月份分表：messages_202601, messages_202602, ...
CREATE TABLE messages_%s (
    id BIGSERIAL,
    msg_id VARCHAR(64) UNIQUE NOT NULL,
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,
    receiver_type VARCHAR(20) NOT NULL, -- 'user' | 'group'
    content TEXT,
    content_type VARCHAR(20) DEFAULT 'text',
    created_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_sender (sender_id),
    INDEX idx_receiver (receiver_id, receiver_type),
    INDEX idx_created (created_at)
);
```

### 3. 用户服务

**职责**：
- 用户注册/登录
- Token 认证
- 用户资料管理
- 好友关系

### 4. 群组服务

**职责**：
- 群创建/解散
- 成员管理
- 权限控制
- 禁言管理

---

## 四、Redis 消息队列设计

### 消息队列结构

```
IM:MQ:Msg:Offline    -> Stream，存储离线消息
IM:MQ:Msg:RealTime   -> Stream，实时消息投递
IM:MQ:Msg:DLQ        -> Stream，死信队列
```

### 消费者组设计

- 100+ 消费者集群
- 批量 IO 优化
- 消费者数量 = CPU 核心数 × 2

---

## 五、API 接口概览

### 认证相关
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| POST | /api/auth/refresh | 刷新 Token |

### 用户相关
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/user/profile | 获取用户资料 |
| PUT | /api/user/profile | 更新用户资料 |
| GET | /api/user/friends | 获取好友列表 |
| POST | /api/user/friend | 添加好友 |

### 消息相关
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/msg/conversations | 获取会话列表 |
| GET | /api/msg/history | 获取聊天记录 |
| POST | /api/msg/send | 发送消息（HTTP） |
| WebSocket | /ws | WebSocket 连接 |

### 群组相关
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/group/create | 创建群组 |
| GET | /api/group/:id | 获取群信息 |
| POST | /api/group/:id/member | 添加成员 |
| DELETE | /api/group/:id/member/:uid | 移除成员 |
| POST | /api/group/:id/mute | 禁言成员 |

---

## 六、WebSocket 消息协议

### 连接流程
```
1. Client -> WS: WSS://server/ws?token=xxx
2. Server: 验证 token，建立连接
3. Server -> Client: {"type": "auth_ok", "user_id": 123}
```

### 消息格式 (JSON)
```json
{
    "type": "message",
    "msg_id": "msg_xxx",
    "from": 123,
    "to": 456,
    "to_type": "user",
    "content": "你好",
    "content_type": "text",
    "timestamp": 1700000000
}
```

### 消息类型
| type | 说明 |
|------|------|
| auth_ok | 认证成功 |
| auth_fail | 认证失败 |
| message | 消息 |
| ack | 消息确认 |
| receipt | 已读回执 |
| recall | 消息撤回 |
| ping/pong | 心跳 |

---

## 七、性能优化策略

### 1. 连接层
- Nginx 4层负载均衡
- WebSocket 连接绑定（sticky session）
- 心跳保活（30s 间隔）

### 2. 消息层
- 批量写入（攒够 100 条或 100ms 批量刷盘）
- 消息分表（按月份）
- 热点数据 Redis 缓存

### 3. 数据库层
- 主从复制读写分离
- 索引优化（覆盖索引）
- 连接池管理

### 4. 缓存层
- 会话信息 Redis 缓存
- 用户信息本地缓存 + Redis
- 群成员列表缓存

---

## 八、部署架构

```yaml
# docker-compose.yml 概要
services:
  nginx:
    image: nginx:latest
    ports: ["80:80", "443:443"]
  
  ws-gateway:
    image: im-ws-gateway:latest
    deploy:
      replicas: 4
  
  api-gateway:
    image: im-api:latest
    deploy:
      replicas: 2
  
  user-service:
    image: im-user:latest
    deploy:
      replicas: 2
  
  msg-service:
    image: im-msg:latest
    deploy:
      replicas: 2
  
  group-service:
    image: im-group:latest
    deploy:
      replicas: 2
  
  redis:
    image: redis:7-alpine
    deploy:
      replicas: 3
  
  postgres:
    image: postgres:15
```

---

## 九、接下来开发计划

1. **Phase 1: 基础骨架** - 项目结构 + 基础 API
2. **Phase 2: 用户模块** - 注册/登录/Token
3. **Phase 3: WebSocket** - 长连接 + 消息收发
4. **Phase 4: 消息服务** - 消息存储 + 已读未读
5. **Phase 5: 群组模块** - 群管理 + 成员权限
6. **Phase 6: 性能优化** - 缓存 + 队列 + 分片
