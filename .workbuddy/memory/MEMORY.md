# IM 后端部署记录

## 服务器信息
- IP: 129.226.74.230
- 目录: /www/wwwroot/im-backend/

## 环境配置
- PostgreSQL: 127.0.0.1:5432, 数据库: im_db, 用户: postgres
- Redis: 127.0.0.1:6379, 密码: Aa147258
- 服务端口: 8080 (HTTP), 8081 (WebSocket)
- JWT_SECRET: p9g1+Xzw4mOKP9qNBKovkTlUcmqIt/SuKj1s6uwl+E=
- JWT_EXPIRE: 168小时

## 启动命令
```bash
cd /www/wwwroot/im-backend
export SERVER_PORT=8080 WS_PORT=8081
export DB_HOST=127.0.0.1 DB_PORT=5432
export DB_USER=postgres DB_PASSWORD=dD2BMcxyebAMmYiM DB_NAME=im_db
export REDIS_HOST=127.0.0.1 REDIS_PORT=6379 REDIS_PASSWORD=Aa147258
export JWT_SECRET=p9g1+Xzw4mOKP9qNBKovkTlUcmqIt/SuKj1s6uwl+E= JWT_EXPIRE=168
nohup ./im-backend-linux > im-backend.log 2>&1 &
```

## 修复的问题
1. 路由重复注册: main.go 中 `/admin/` 被注册两次，删除重复行后修复
2. Redis 密码: Aa147258 (之前漏了)
3. Go 版本: 使用服务器上的 Go 1.25 编译

## 服务状态
- 状态: ✅ 运行中 (2026-03-21 00:10)
- Health: http://129.226.74.230:8080/health
