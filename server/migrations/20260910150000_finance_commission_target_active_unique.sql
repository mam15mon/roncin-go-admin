DROP INDEX IF EXISTS "financecommission_active_source_employee_rule";

CREATE UNIQUE INDEX "finance_commissions_target_active_unique"
  ON "finance_commissions"("organization_id", "verification_id", "employee_id", "personnel_role")
  WHERE "status" <> 'CANCELLED';
