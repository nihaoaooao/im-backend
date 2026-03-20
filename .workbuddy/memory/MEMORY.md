# IM 后端部署记录

## 服务器信息
- IP: 129.226.74.230
- 目录: /www/wwwroot/im-backend/

## 环境配置
- PostgreSQL: 127.0.0.1:35432, 数据库: im_db, 用户: postgres
- Redis: 127.0.0.1:6379
- 服务端口: 8080 (HTTP), 8081 (WebSocket)
- JWT_SECRET: p9g1+Xzw4mOKP9qNBKovkTlUcmqIt/SuKj1s6uwl+E=
- JWT_EXPIRE: 168小时

## 部署过程问题与解决
1. 数据库密码修正: dD2BMcxyebAWmYiM → dD2BMcxyebAMmYiM
2. 添加唯一约束 uni_users_username: 使用 Python psycopg2 执行 SQL
3. 添加唯一约束 uni_user_tokens_token: 使用 Python psycopg2 执行 SQL
4. GORM 删除不存在的约束: 删除旧的 users_username_key 约束

## 启动命令
```bash
cd /www/wwwroot/im-backend
export SERVER_PORT=8080 WS_PORT=8081 DB_HOST=127.0.0.1 DB_PORT=35432 DB_USER=postgres DB_PASSWORD=dD2BMcxyebAMmYiM DB_NAME=im_db REDIS_HOST=127.0.0.1 REDIS_PORT=6379 JWT_SECRET=p9g1+Xzw4mOKP9qNBKovkTlUcmqIt/SuKj1s6uwl+E= JWT_EXPIRE=168
nohup ./im-backend-linux > im-backend.log 2>&1 &
```

## 服务状态
- 状态: ✅ 运行中
- Health: http://129.226.74.230:8080/health
