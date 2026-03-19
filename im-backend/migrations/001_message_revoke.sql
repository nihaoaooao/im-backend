-- =====================================================
-- 消息撤回功能数据库迁移
-- 添加 revoked、revoked_at、can_revoke 字段
-- =====================================================

-- 1. 为消息表添加撤回相关字段
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS revoked BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS can_revoke BOOLEAN DEFAULT TRUE;

-- 2. 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_messages_is_recalled ON messages(is_recalled);
CREATE INDEX IF NOT EXISTS idx_messages_revoked ON messages(revoked);
CREATE INDEX IF NOT EXISTS idx_messages_can_revoke ON messages(can_revoke);
CREATE INDEX IF NOT EXISTS idx_messages_sender_time ON messages(sender_id, created_at DESC);

-- 3. 更新现有分表函数（如果有）
CREATE OR REPLACE FUNCTION create_message_table_if_not_exists(year_month TEXT)
RETURNS VOID AS $$
DECLARE
    table_name TEXT := 'messages_' || year_month;
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I (
            id BIGSERIAL PRIMARY KEY,
            msg_id VARCHAR(64) UNIQUE NOT NULL,
            conversation_id BIGINT NOT NULL,
            sender_id BIGINT NOT NULL,
            receiver_id BIGINT NOT NULL,
            receiver_type VARCHAR(20) NOT NULL,
            content TEXT,
            content_type VARCHAR(20) DEFAULT ''text'',
            extra JSONB,
            is_recalled BOOLEAN DEFAULT FALSE,
            revoked BOOLEAN DEFAULT FALSE,
            revoked_at TIMESTAMP,
            can_revoke BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )', table_name
    );

    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_conversation ON %I(conversation_id)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_sender ON %I(sender_id)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_receiver ON %I(receiver_id, receiver_type)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_time ON %I(created_at)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_is_recalled ON %I(is_recalled)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_revoked ON %I(revoked)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_can_revoke ON %I(can_revoke)', table_name, table_name);
END;
$$ LANGUAGE plpgsql;

-- 4. 创建消息撤回记录表（可选，用于审计）
CREATE TABLE IF NOT EXISTS message_recalls (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    msg_id VARCHAR(64) NOT NULL,
    conversation_id BIGINT NOT NULL,
    sender_id BIGINT NOT NULL,
    revoker_id BIGINT NOT NULL,
    recall_time TIMESTAMP DEFAULT NOW(),
    reason VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_message_recalls_message ON message_recalls(message_id);
CREATE INDEX IF NOT EXISTS idx_message_recalls_sender ON message_recalls(sender_id);
CREATE INDEX IF NOT EXISTS idx_message_recalls_time ON message_recalls(recall_time DESC);
