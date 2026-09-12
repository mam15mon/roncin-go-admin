-- 提成来源二选一：核销（verification_id）或对冲（netting_id），恰好一个非空。
-- 存量核销来源行不受影响；新增对冲来源列为可空并保留外键参照。
ALTER TABLE "finance_commissions"
  ALTER COLUMN "verification_id" DROP NOT NULL,
  ALTER COLUMN "verification_no" DROP NOT NULL;

ALTER TABLE "finance_commissions"
  ADD COLUMN "netting_id" uuid REFERENCES "finance_nettings"("id"),
  ADD COLUMN "netting_no" varchar(64);

-- 活跃去重唯一约束按来源拆分为两个部分唯一索引：
-- 同来源同员工同角色仅一条非终态提成；来源为空的行不参与任何一侧。
DROP INDEX IF EXISTS "finance_commissions_target_active_unique";
CREATE UNIQUE INDEX "finance_commissions_verification_active_unique"
  ON "finance_commissions"("organization_id","verification_id","employee_id","personnel_role")
  WHERE "status" <> 'CANCELLED' AND "verification_id" IS NOT NULL;
CREATE UNIQUE INDEX "finance_commissions_netting_active_unique"
  ON "finance_commissions"("organization_id","netting_id","employee_id","personnel_role")
  WHERE "status" <> 'CANCELLED' AND "netting_id" IS NOT NULL;

-- 来源检索索引改为部分索引（各自来源非空），并为对冲来源补同构索引。
DROP INDEX IF EXISTS "financecommission_source_employee_status";
DROP INDEX IF EXISTS "financecommission_verification_id_employee_id_status";
CREATE INDEX "financecommission_source_employee_status"
  ON "finance_commissions"("verification_id","employee_id","status")
  WHERE "verification_id" IS NOT NULL;
CREATE INDEX "financecommission_netting_employee_status"
  ON "finance_commissions"("netting_id","employee_id","status")
  WHERE "netting_id" IS NOT NULL;

-- 调整单来源类型新增对冲反转；既有取值不受影响。
ALTER TABLE "finance_commission_adjustments"
  DROP CONSTRAINT IF EXISTS "commission_adjustment_source_type_check";
ALTER TABLE "finance_commission_adjustments"
  ADD CONSTRAINT "commission_adjustment_source_type_check"
  CHECK("source_type" IN('MANUAL','VERIFICATION_REVERSAL','NETTING_REVERSAL'));
