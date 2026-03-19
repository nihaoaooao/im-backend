-- =====================================================
-- 壹信 IM 数据库迁移脚本
-- 版本: 001
-- 描述: 初始表结构
-- =====================================================

-- -----------------------------------------------------
-- 1. 用户表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    phone VARCHAR(20),
    nickname VARCHAR(100),
    avatar VARCHAR(255),
    gender VARCHAR(10) DEFAULT 'unknown',
    status VARCHAR(20) DEFAULT 'active',
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- -----------------------------------------------------
-- 2. 会话表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(20) NOT NULL,
    name VARCHAR(100),
    creator_id BIGINT,
    last_msg_id BIGINT,
    last_msg_content TEXT,
    last_msg_time TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversations_type ON conversations(type);
CREATE INDEX idx_conversations_creator ON conversations(creator_id);

-- -----------------------------------------------------
-- 3. 消息表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    msg_id VARCHAR(64) NOT NULL UNIQUE,
    conversation_id BIGINT NOT NULL,
    sender_id BIGINT NOT NULL,
    content TEXT,
    content_type VARCHAR(20) DEFAULT 'text',
    extra JSONB,
    is_recalled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_created ON messages(created_at);

-- -----------------------------------------------------
-- 4. 会话成员表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS conversation_members (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(20) DEFAULT 'member',
    nickname VARCHAR(100),
    unread_count BIGINT DEFAULT 0,
    is_muted BOOLEAN DEFAULT FALSE,
    mute_until TIMESTAMP,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (conversation_id, user_id)
);

CREATE INDEX idx_conversation_members_user ON conversation_members(user_id);

-- -----------------------------------------------------
-- 5. 已读回执表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS message_reads (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    conversation_id BIGINT NOT NULL,
    read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (message_id, user_id)
);

CREATE INDEX idx_message_reads_user ON message_reads(user_id);
CREATE INDEX idx_message_reads_conversation ON message_reads(conversation_id);

-- -----------------------------------------------------
-- 6. 好友关系表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS friendships (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    friend_id BIGINT NOT NULL,
    remark VARCHAR(100),
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, friend_id)
);

CREATE INDEX idx_friendships_user ON friendships(user_id);
CREATE INDEX idx_friendships_friend ON friendships(friend_id);

-- -----------------------------------------------------
-- 7. 用户 Token 表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS user_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    device_type VARCHAR(50),
    device_id VARCHAR(100),
    ip VARCHAR(50),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_tokens_token ON user_tokens(token);
CREATE INDEX idx_user_tokens_user ON user_tokens(user_id);
CREATE INDEX idx_user_tokens_expires ON user_tokens(expires_at);

-- =====================================================
-- 初始化数据
-- =====================================================

-- 插入管理员用户 (密码: admin123，使用 bcrypt 加密)
INSERT INTO users (username, password_hash, nickname, status) VALUES
('admin', '$2a$10$N9qo8LOickgx2ZMRZoMyeIjZRGdjGj/n3.zS3LPOW1j/C.P5RVkYu', '系统管理员', 'active')
ON CONFLICT (username) DO NOTHING;
