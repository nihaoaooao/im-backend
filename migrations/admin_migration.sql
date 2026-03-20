-- 管理后台数据库迁移脚本
-- 执行时间: 2026-03-20
-- 用途: 为 IM 系统添加管理后台所需的数据库表和字段

-- 1. 修改 users 表，添加 login_count 字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS login_count INT DEFAULT 0;

-- 2. 创建管理操作日志表
CREATE TABLE IF NOT EXISTS admin_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    admin_id BIGINT NOT NULL COMMENT '管理员ID',
    admin_username VARCHAR(50) NOT NULL COMMENT '管理员用户名',
    action VARCHAR(50) NOT NULL COMMENT '操作类型: ban_user, unban_user, dismiss_group, revoke_message 等',
    target_type VARCHAR(50) NOT NULL COMMENT '目标类型: user, group, message',
    target_id BIGINT NOT NULL COMMENT '目标ID',
    details TEXT COMMENT '操作详情',
    ip VARCHAR(50) COMMENT '操作IP',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_admin (admin_id),
    INDEX idx_target (target_type, target_id),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理操作日志表';

-- 3. 创建敏感词表
CREATE TABLE IF NOT EXISTS sensitive_words (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    word VARCHAR(100) NOT NULL UNIQUE COMMENT '敏感词',
    type ENUM('text', 'image', 'file') DEFAULT 'text' COMMENT '敏感词类型',
    level ENUM('warning', 'block') DEFAULT 'block' COMMENT '处理级别: warning=警告, block=拦截',
    status ENUM('active', 'inactive') DEFAULT 'active' COMMENT '状态',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_word (word),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='敏感词表';

-- 4. 初始化一个超级管理员用户 (密码: admin123, 需要在生产环境修改)
-- 注意: 这是明文密码的hash示例，实际使用请用 bcrypt 生成
-- INSERT INTO users (username, password_hash, role, email, status, created_at)
-- VALUES ('admin', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 'super_admin', 'admin@example.com', 'active', NOW());
