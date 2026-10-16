-- 旧 self 范围从未提供有效授权，先停用其角色，避免退役范围配置后恢复授权。
UPDATE "roles" SET "enabled" = false, "updated_at" = now() WHERE "data_scope" = 'self';
-- 角色仅配置功能权限；数据边界由公司/系统工作台推导。
ALTER TABLE "roles" DROP COLUMN "data_scope";
