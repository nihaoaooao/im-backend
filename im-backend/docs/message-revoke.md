# 消息撤回功能实现

## 功能概述

消息撤回功能允许用户在发送消息后的2分钟内撤回自己发送的消息。

## 数据库变更

### 迁移文件
- `migrations/001_message_revoke.sql`

### 新增字段
- `revoked`: 是否已撤回
- `revoked_at`: 撤回时间
- `can_revoke`: 是否可以撤回

### 新增表
- `message_recalls`: 消息撤回记录表

## API接口

### 撤回消息
- **POST** `/api/v1/messages/recall`
- **请求体**: `{"msg_id": "消息ID"}`
- **响应**: `{"code": 200, "message": "消息撤回成功", "msg_id": "消息ID"}`

### 获取可撤回消息
- **GET** `/api/v1/messages/recallable?limit=50`
- **响应**: 返回当前用户发送的可在撤回时限内的消息列表

## 核心逻辑

### 撤回规则
1. 只能撤回自己发送的消息
2. 需在2分钟内撤回
3. 消息只能撤回一次

### 撤回流程
1. 验证用户身份
2. 检查消息是否存在
3. 检查是否是消息发送者
4. 检查是否在撤回时限内（2分钟）
5. 执行撤回（更新数据库）
6. 记录撤回日志
7. 广播撤回通知给相关用户

### 定时任务
- 每分钟检查一次超过撤回时限的消息
- 将这些消息标记为不可撤回

## WebSocket通知

撤回成功后会通过WebSocket向相关用户发送撤回通知：
```json
{
  "type": "recall",
  "msg_id": "消息ID",
  "conversation_id": "会话ID",
  "revoker_id": "撤回者ID",
  "original_sender_id": "原始发送者ID",
  "revoked_at": 撤回时间戳
}
```

## 单元测试

运行测试：
```bash
go test -v ./service/...
go test -v ./utils/...
```

## 配置项

环境变量：
- `SERVER_PORT`: 服务器端口（默认8080）
- `SERVER_READ_TIMEOUT`: 读取超时（默认60s）
- `SERVER_WRITE_TIMEOUT`: 写入超时（默认60s）
