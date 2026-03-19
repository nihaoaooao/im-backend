-- =====================================================
-- 多媒体消息上传数据库迁移
-- 创建 media 表用于存储多媒体文件信息
-- =====================================================

-- 1. 创建多媒体文件表
CREATE TABLE IF NOT EXISTS media (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL, -- image, voice, video
    original_url VARCHAR(500) NOT NULL,
    thumbnail_url VARCHAR(500),
    file_size BIGINT NOT NULL,
    width INT,
    height INT,
    duration DECIMAL(10,2), -- 语音/视频时长（秒）
    format VARCHAR(20),
    metadata JSONB, -- 额外元数据（EXIF信息等）
    storage_key VARCHAR(500) NOT NULL, -- 对象存储Key
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- 索引
    CONSTRAINT uk_media_storage UNIQUE (storage_key)
);

-- 2. 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_media_user ON media(user_id);
CREATE INDEX IF NOT EXISTS idx_media_type ON media(type);
CREATE INDEX IF NOT EXISTS idx_media_created ON media(created_at DESC);

-- 3. 用户文件配额表（可选）
CREATE TABLE IF NOT EXISTS user_media_quota (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    total_size BIGINT DEFAULT 0, -- 已使用存储空间（字节）
    file_count BIGINT DEFAULT 0, -- 已上传文件数量
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_quota_user ON user_media_quota(user_id);
