-- 客户收款约定收口：先验证正额度币种，再清理与合并旧规则。
DO $$
DECLARE bad_count bigint;
BEGIN
  -- 提前取得后续删列所需的表锁，守卫通过后旧程序不能再写入未校验规则。
  LOCK TABLE "partner_settlement_rules" IN ACCESS EXCLUSIVE MODE;
  SELECT count(*) INTO bad_count
  FROM "partner_settlement_rules" r
  JOIN "partner_roles" pr ON pr.id = r.partner_role_id
  JOIN "partners" p ON p.id = pr.partner_id
  JOIN "organizations" o ON o.id = p.organization_id
  WHERE pr.role_type = 'customer' AND r.is_active AND r.credit_limit_minor > 0
    AND (o.base_currency IS NULL OR r.credit_currency IS NULL OR r.credit_currency <> o.base_currency);
  IF bad_count > 0 THEN
    RAISE EXCEPTION '收款约定迁移中止：% 条启用的客户结算规则的信用额度缺少币种或不是公司本币，请先在客户档案中修正额度后再迁移', bad_count;
  END IF;
END $$;

DELETE FROM "partner_settlement_rules" r
USING "partner_roles" pr
WHERE r.partner_role_id = pr.id AND pr.role_type <> 'customer';

DELETE FROM "partner_settlement_rules" WHERE NOT is_active;

-- 保留最近更新的行，按旧业务实际采用的最大账期和最大正额度合并。
-- 全组没有正额度时，保留行原额度与币种；只有重复组才发生更新。
WITH ranked AS (
  SELECT id, partner_role_id,
         row_number() OVER (PARTITION BY partner_role_id ORDER BY updated_at DESC, id DESC) AS position,
         count(*) OVER (PARTITION BY partner_role_id) AS group_count,
         max(payment_terms_days) OVER (PARTITION BY partner_role_id) AS merged_terms
  FROM "partner_settlement_rules"
), best_credit AS (
  SELECT DISTINCT ON (partner_role_id) partner_role_id, credit_limit_minor, credit_currency
  FROM "partner_settlement_rules"
  WHERE credit_limit_minor > 0
  ORDER BY partner_role_id, credit_limit_minor DESC, updated_at DESC, id DESC
)
UPDATE "partner_settlement_rules" kept
SET payment_terms_days = ranked.merged_terms,
    credit_limit_minor = CASE WHEN best_credit.partner_role_id IS NOT NULL THEN best_credit.credit_limit_minor ELSE kept.credit_limit_minor END,
    credit_currency = CASE WHEN best_credit.partner_role_id IS NOT NULL THEN best_credit.credit_currency ELSE kept.credit_currency END
FROM ranked
LEFT JOIN best_credit ON best_credit.partner_role_id = ranked.partner_role_id
WHERE kept.id = ranked.id AND ranked.position = 1 AND ranked.group_count > 1;

WITH ranked AS (
  SELECT id, row_number() OVER (PARTITION BY partner_role_id ORDER BY updated_at DESC, id DESC) AS position
  FROM "partner_settlement_rules"
)
DELETE FROM "partner_settlement_rules" removed
USING ranked
WHERE removed.id = ranked.id AND ranked.position > 1;

DROP INDEX "partner_settlement_rule_key";
DROP INDEX "partnersettlementrule_partner_role_id_is_active";
ALTER TABLE "partner_settlement_rules"
  DROP COLUMN "statement_mode",
  DROP COLUMN "settlement_method",
  DROP COLUMN "settlement_day",
  DROP COLUMN "settlement_cycle_days",
  DROP COLUMN "settlement_base",
  DROP COLUMN "settlement_currency",
  DROP COLUMN "is_active";

CREATE UNIQUE INDEX "partner_settlement_rule_role_key" ON "partner_settlement_rules" ("partner_role_id");
