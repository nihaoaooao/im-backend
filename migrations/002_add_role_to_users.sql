-- PT-002: 权限验证修复 - 添加用户角色字段
-- 迁移时间：2026-03-27

-- 添加 role 字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- 为现有用户设置默认角色
UPDATE users SET role = 'user' WHERE role IS NULL OR role = '';

-- 将第一个用户设置为管理员（可选，根据实际情况调整）
-- UPDATE users SET role = 'admin' WHERE id = 1;

-- 添加注释
COMMENT ON COLUMN users.role IS '用户角色: admin=管理员, user=普通用户';
