-- 取消登录偏好配置，保留成员关系与角色绑定。由 Ent 字段差异经 Atlas PostgreSQL 规划生成。
ALTER TABLE "memberships" DROP COLUMN "primary";
