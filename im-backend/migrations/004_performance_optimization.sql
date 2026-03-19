-- =====================================================
-- 性能优化数据库迁移
-- 添加索引优化查询性能
-- =====================================================

-- 1. 消息表复合索引优化
-- 查询发送者消息列表
CREATE INDEX IF NOT EXISTS idx_messages_sender_conversation ON messages(sender_id, conversation_id DESC);

-- 查询接收者消息列表
CREATE INDEX IF NOT EXISTS idx_messages_receiver_conversation ON messages(receiver_id, receiver_type, conversation_id DESC);

-- 查询会话消息（最常用）
CREATE INDEX IF NOT EXISTS idx_messages_conversation_time ON messages(conversation_id, created_at DESC);

-- 查询特定时间范围消息
CREATE INDEX IF NOT EXISTS idx_messages_time_range ON messages(created_at DESC) WHERE created_at > NOW() - INTERVAL '7 days';

-- 2. 已读记录表索引优化
-- 查询用户已读消息
CREATE INDEX IF NOT EXISTS idx_message_reads_user_time ON message_reads(user_id, read_at DESC);

-- 3. 会话表索引优化（如果存在）
-- CREATE INDEX IF NOT EXISTS idx_conversations_user ON conversations(user_id, updated_at DESC);
-- CREATE INDEX IF NOT EXISTS idx_conversations_type ON conversations(type, updated_at DESC);

-- 4. 用户表索引优化
-- CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
-- CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
-- CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- 5. 好友表索引优化
-- CREATE INDEX IF NOT EXISTS idx_friends_user ON friends(user_id, status);
-- CREATE INDEX IF NOT EXISTS idx_friends_friend ON friends(friend_id, status);

-- 6. 统计表（用于缓存热点数据）
-- CREATE TABLE IF NOT EXISTS message_stats (
--     id BIGSERIAL PRIMARY KEY,
--     conversation_id BIGINT NOT NULL,
--     message_count BIGINT DEFAULT 0,
--     last_message_at TIMESTAMP,
--     updated_at TIMESTAMP DEFAULT NOW()
-- );
-- CREATE INDEX IF NOT EXISTS idx_message_stats_conversation ON message_stats(conversation_id);

-- 7. 在线用户缓存表
-- CREATE TABLE IF NOT EXISTS online_users (
--     id BIGSERIAL PRIMARY KEY,
--     user_id BIGINT NOT NULL UNIQUE,
--     last_active_at TIMESTAMP DEFAULT NOW(),
--     device_info VARCHAR(255)
-- );
-- CREATE INDEX IF NOT EXISTS idx_online_users_active ON online_users(last_active_at DESC);

-- 8. 清理过期数据（可选，定期执行）
-- DELETE FROM message_reads WHERE read_at < NOW() - INTERVAL '30 days';
-- VACUUM ANALYZE message_reads;
