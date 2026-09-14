-- 客商代码改为选填：留空表示未设置，服务端不再自动生成。
-- 唯一索引 partner_org_code_key 依赖 PostgreSQL 默认 NULLS DISTINCT，
-- 允许多行代码为 NULL，显式重复代码仍被约束拦截。

ALTER TABLE "partners" ALTER COLUMN "code" DROP NOT NULL;
