-- 收款约定迁移前的只读盘点（不修改任何数据）。
-- 用法（在仓库根目录，按 .env.local 的 DATABASE_SOURCE 连接开发库）：
--   psql "$DATABASE_SOURCE" -v ON_ERROR_STOP=1 -f .trellis/tasks/11-18-customer-payment-terms/research/preflight-inventory.sql
-- 脚本只创建会话级临时视图，最后 ROLLBACK，不写入任何业务数据；
-- （PostgreSQL 的只读事务不允许 CREATE TEMP VIEW，所以这里用普通事务加回滚。）
-- 把输出整理给用户确认后，才能执行迁移。
-- 本脚本按规划时的字段写成；如果实施前 Schema 已有变化，先对照修正再运行。

BEGIN;

CREATE TEMP VIEW pt_rules AS
SELECT r.id AS rule_id, r.partner_role_id, pr.role_type, r.is_active,
       r.payment_terms_days, r.credit_limit_minor, r.credit_currency, r.updated_at,
       p.id AS partner_id, p.code AS partner_code, p.legal_name, p.organization_id,
       o.kind AS org_kind, o.base_currency AS org_base_currency
FROM partner_settlement_rules r
JOIN partner_roles pr ON pr.id = r.partner_role_id
JOIN partners p ON p.id = pr.partner_id
LEFT JOIN organizations o ON o.id = p.organization_id;

-- A. 总览
SELECT
  count(*)                                                        AS total_rules,
  count(*) FILTER (WHERE role_type <> 'customer')                 AS delete_non_customer,
  count(*) FILTER (WHERE role_type = 'customer' AND NOT is_active) AS delete_inactive_customer,
  count(*) FILTER (WHERE role_type = 'customer' AND is_active)    AS active_customer
FROM pt_rules;

-- B. 将被删除的非客户角色规则（逐条）
SELECT rule_id, partner_code, legal_name, role_type, is_active, payment_terms_days, credit_limit_minor, credit_currency
FROM pt_rules WHERE role_type <> 'customer'
ORDER BY legal_name, role_type;

-- C. 将被删除的停用客户规则（逐条）
SELECT rule_id, partner_code, legal_name, payment_terms_days, credit_limit_minor, credit_currency, updated_at
FROM pt_rules WHERE role_type = 'customer' AND NOT is_active
ORDER BY legal_name, updated_at DESC;

-- D. 同一客户角色有多条启用规则：合并前明细 + 合并后结果
WITH active AS (
  SELECT * FROM pt_rules WHERE role_type = 'customer' AND is_active
), dup AS (
  SELECT partner_role_id FROM active GROUP BY partner_role_id HAVING count(*) > 1
)
SELECT a.partner_role_id, a.partner_code, a.legal_name, a.rule_id, a.updated_at,
       a.payment_terms_days, a.credit_limit_minor, a.credit_currency,
       (row_number() OVER (PARTITION BY a.partner_role_id ORDER BY a.updated_at DESC, a.rule_id DESC) = 1) AS kept,
       max(a.payment_terms_days) OVER (PARTITION BY a.partner_role_id) AS merged_terms,
       max(a.credit_limit_minor) FILTER (WHERE a.credit_limit_minor > 0) OVER (PARTITION BY a.partner_role_id) AS merged_credit
FROM active a JOIN dup d USING (partner_role_id)
ORDER BY a.legal_name, a.updated_at DESC;

-- E. 额度币种异常（迁移守卫会因这些记录直接失败，需先人工修正）
--    口径与 partnerCreditSummaries 一致：只看启用的客户规则，且额度 > 0。
SELECT rule_id, partner_code, legal_name, credit_limit_minor, credit_currency, org_base_currency,
       CASE
         WHEN org_base_currency IS NULL THEN '往来单位不属于公司（组织无本币）'
         WHEN credit_currency IS NULL THEN '缺少额度币种'
         ELSE '额度币种不是公司本币'
       END AS problem
FROM pt_rules
WHERE role_type = 'customer' AND is_active AND credit_limit_minor > 0
  AND (org_base_currency IS NULL OR credit_currency IS NULL OR credit_currency <> org_base_currency)
ORDER BY legal_name;

-- F. 额度为 0 的启用客户规则（当前信用判断把 0 视为不限额，迁移不改动其取值，只列出供知悉）
SELECT rule_id, partner_code, legal_name, credit_limit_minor, credit_currency
FROM pt_rules
WHERE role_type = 'customer' AND is_active AND credit_limit_minor = 0
ORDER BY legal_name;

ROLLBACK;
