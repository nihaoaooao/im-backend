-- =====================================================
-- 壹信 IM 数据库设计
-- 数据库: im_db
-- =====================================================

-- -----------------------------------------------------
-- 1. 用户表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    avatar VARCHAR(500),
    phone VARCHAR(20),
    email VARCHAR(255),
    gender VARCHAR(10) DEFAULT 'unknown',
    status VARCHAR(20) DEFAULT 'active', -- active, banned
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- -----------------------------------------------------
-- 2. 用户 Token 表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS user_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    device_type VARCHAR(50), -- web, ios, android
    device_id VARCHAR(100),
    ip VARCHAR(50),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tokens_token ON user_tokens(token);
CREATE INDEX idx_tokens_user ON user_tokens(user_id);
CREATE INDEX idx_tokens_expires ON user_tokens(expires_at);

-- -----------------------------------------------------
-- 3. 好友关系表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS friendships (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    remark VARCHAR(100), -- 好友备注
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, blocked
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, friend_id)
);

CREATE INDEX idx_friendships_user ON friendships(user_id);
CREATE INDEX idx_friendships_friend ON friendships(friend_id);
CREATE INDEX idx_friendships_status ON friendships(status);

-- -----------------------------------------------------
-- 4. 会话表 (一对一的私聊)
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    conversation_type VARCHAR(20) NOT NULL, -- private, group
    user_id_1 BIGINT NOT NULL, -- 用户1
    user_id_2 BIGINT, -- 用户2 (私聊时)
    group_id BIGINT, -- 群ID (群聊时)
    last_msg_id BIGINT, -- 最后一条消息ID
    last_msg_content TEXT, -- 最后一条消息摘要
    last_msg_time TIMESTAMP, -- 最后消息时间
    unread_count_1 BIGINT DEFAULT 0, -- 用户1未读数
    unread_count_2 BIGINT DEFAULT 0, -- 用户2未读数
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id_1, user_id_2),
    UNIQUE(group_id)
);

CREATE INDEX idx_conversations_user1 ON conversations(user_id_1);
CREATE INDEX idx_conversations_user2 ON conversations(user_id_2);
CREATE INDEX idx_conversations_group ON conversations(group_id);
CREATE INDEX idx_conversations_time ON conversations(last_msg_time DESC);

-- -----------------------------------------------------
-- 5. 消息表 (按月份分表)
-- -----------------------------------------------------
-- 注意: 实际使用时创建 messages_YYYYMM 格式的分表
-- 这里创建基础表结构模板
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    msg_id VARCHAR(64) UNIQUE NOT NULL, -- 消息唯一ID (雪花算法)
    conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,
    receiver_type VARCHAR(20) NOT NULL, -- user, group
    content TEXT,
    content_type VARCHAR(20) DEFAULT 'text', -- text, image, audio, video, file
    extra JSONB, -- 扩展信息
    is_recalled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 索引 (每个分表都需要)
-- CREATE INDEX idx_messages_conversation ON messages(conversation_id);
-- CREATE INDEX idx_messages_sender ON messages(sender_id);
-- CREATE INDEX idx_messages_receiver ON messages(receiver_id, receiver_type);
-- CREATE INDEX idx_messages_time ON messages(created_at);

-- -----------------------------------------------------
-- 6. 消息已读状态表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS message_reads (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    conversation_id BIGINT NOT NULL,
    read_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);

CREATE INDEX idx_message_reads_user ON message_reads(user_id);
CREATE INDEX idx_message_reads_conversation ON message_reads(conversation_id);

-- -----------------------------------------------------
-- 7. 群组表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    group_id VARCHAR(64) UNIQUE NOT NULL, -- 群ID
    name VARCHAR(100) NOT NULL,
    avatar VARCHAR(500),
    owner_id BIGINT NOT NULL, -- 群主
    description TEXT,
    member_count INT DEFAULT 0,
    max_members INT DEFAULT 500,
    mute_all BOOLEAN DEFAULT FALSE, -- 全员禁言
    status VARCHAR(20) DEFAULT 'active', -- active, dissolved
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_groups_group_id ON groups(group_id);
CREATE INDEX idx_groups_owner ON groups(owner_id);
CREATE INDEX idx_groups_status ON groups(status);

-- -----------------------------------------------------
-- 8. 群成员表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member', -- owner, admin, member
    nickname VARCHAR(100), -- 群昵称
    is_muted BOOLEAN DEFAULT FALSE, -- 是否被禁言
    mute_until TIMESTAMP, -- 禁言截止时间
    joined_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_group_members_group ON group_members(group_id);
CREATE INDEX idx_group_members_user ON group_members(user_id);
CREATE INDEX idx_group_members_role ON group_members(role);

-- -----------------------------------------------------
-- 9. 群邀请/申请表
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS group_applications (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    inviter_id BIGINT, -- 邀请人
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, rejected
    message VARCHAR(255), -- 验证消息
    created_at TIMESTAMP DEFAULT NOW(),
    handled_at TIMESTAMP
);

CREATE INDEX idx_group_applications_group ON group_applications(group_id);
CREATE INDEX idx_group_applications_user ON group_applications(user_id);

-- =====================================================
-- 初始化数据
-- =====================================================

-- 插入管理员用户 (密码: admin123)
INSERT INTO users (username, password_hash, nickname, role, status) VALUES
('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMye/U.N4.5F.HQW5R.HGhM7v/2fHdywJSe', '系统管理员', 'admin', 'active')
ON CONFLICT (username) DO NOTHING;

-- =====================================================
-- 分表函数 (消息表按月分表)
-- =====================================================

-- 创建消息分表的函数
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
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )', table_name
    );

    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_conversation ON %I(conversation_id)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_sender ON %I(sender_id)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_receiver ON %I(receiver_id, receiver_type)', table_name, table_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_time ON %I(created_at)', table_name, table_name);
END;
$$ LANGUAGE plpgsql;
