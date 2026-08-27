-- 保留核销已实现毛利口径，将人员归属由订单人员切换为客户档案人员。
-- 当前仍处于开发阶段，旧提成快照无法补齐客户与费用快照，显式清理后重建。
DELETE FROM "finance_commission_adjustments";
DELETE FROM "finance_commission_lines";
DELETE FROM "finance_commissions";
DELETE FROM "finance_commission_rules";

DROP INDEX IF EXISTS "financecommission_active_source_employee_rule";
DROP INDEX IF EXISTS "financecommission_active_source_employee";
DROP INDEX IF EXISTS "financecommission_source_employee_status";

ALTER TABLE "finance_commission_rules"
  DROP CONSTRAINT IF EXISTS "commission_rule_role_check",
  DROP CONSTRAINT IF EXISTS "commission_rule_basis_check",
  ADD CONSTRAINT "commission_rule_role_check" CHECK("personnel_role" IN('SALES','OPERATOR','CUSTOMER_SERVICE')),
  ADD CONSTRAINT "commission_rule_basis_check" CHECK("calculation_basis" IN('REALIZED_PROFIT','REALIZED_REVENUE'));

ALTER TABLE "finance_commissions"
  DROP CONSTRAINT IF EXISTS "finance_commission_rule_snapshot_check",
  ALTER COLUMN "calculation_version" SET DEFAULT 'CUSTOMER_REALIZED_PROFIT_V2',
  ADD COLUMN "customer_count" integer NOT NULL,
  ADD COLUMN "order_count" integer NOT NULL,
  ADD COLUMN "fee_count" integer NOT NULL,
  ADD COLUMN "commission_base_amount" numeric(28,8) NOT NULL,
  ADD CONSTRAINT "finance_commission_rule_snapshot_check" CHECK(
    "rule_id" IS NOT NULL AND "rule_name" IS NOT NULL
    AND "personnel_role" IN('SALES','OPERATOR','CUSTOMER_SERVICE')
    AND "calculation_basis" IN('REALIZED_PROFIT','REALIZED_REVENUE')
  ),
  ADD CONSTRAINT "finance_commission_counts_check" CHECK("customer_count">=0 AND "order_count">=0 AND "fee_count">=0),
  ADD CONSTRAINT "finance_commission_base_nonnegative" CHECK("commission_base_amount">=0);

CREATE UNIQUE INDEX "financecommission_active_source_employee_rule"
  ON "finance_commissions"("verification_id","employee_id","rule_id")
  WHERE "status"<>'CANCELLED';
CREATE INDEX "financecommission_source_employee_status"
  ON "finance_commissions"("verification_id","employee_id","status");

ALTER TABLE "finance_commission_lines"
  DROP CONSTRAINT IF EXISTS "commission_line_role_check",
  DROP CONSTRAINT IF EXISTS "commission_line_basis_check",
  ADD COLUMN "order_date" varchar(32) NOT NULL,
  ADD COLUMN "customer_id" uuid NOT NULL,
  ADD COLUMN "customer_code" varchar(64) NOT NULL,
  ADD COLUMN "customer_name" varchar(200) NOT NULL,
  ADD COLUMN "fee_count" integer NOT NULL,
  ADD COLUMN "fee_snapshot" jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN "commission_base_amount" numeric(28,8) NOT NULL,
  ADD CONSTRAINT "commission_line_role_check" CHECK("personnel_role" IN('SALES','OPERATOR','CUSTOMER_SERVICE')),
  ADD CONSTRAINT "commission_line_basis_check" CHECK("calculation_basis" IN('REALIZED_PROFIT','REALIZED_REVENUE')),
  ADD CONSTRAINT "commission_line_fee_count_check" CHECK("fee_count">=0),
  ADD CONSTRAINT "commission_line_base_nonnegative" CHECK("commission_base_amount">=0);

CREATE INDEX "financecommissionline_customer" ON "finance_commission_lines"("organization_id","customer_id");
