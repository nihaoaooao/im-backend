-- =====================================================
-- 已读回执功能数据库迁移
-- 添加 message_reads 表用于存储已读状态
-- =====================================================

-- 1. 创建消息已读记录表
CREATE TABLE IF NOT EXISTS message_reads (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    read_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- 唯一约束：每个用户对每条消息只能有一条已读记录
    CONSTRAINT uk_message_user UNIQUE (message_id, user_id)
);

-- 2. 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_message_reads_message ON message_reads(message_id);
CREATE INDEX IF NOT EXISTS idx_message_reads_user ON message_reads(user_id);
CREATE INDEX IF NOT EXISTS idx_message_reads_time ON message_reads(read_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_reads_message_user ON message_reads(message_id, user_id);

-- 3. 创建会话未读数表（可选，用于缓存）
CREATE TABLE IF NOT EXISTS conversation_unread_counts (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    unread_count BIGINT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT uk_conversation_user UNIQUE (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_unread_conversation ON conversation_unread_counts(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_unread_user ON conversation_unread_counts(user_id);

-- 4. 为 messages 表添加最后阅读时间字段（可选，用于优化）
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS last_read_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_messages_last_read ON messages(last_read_at);
