DROP INDEX "financecommission_source_employee";
CREATE UNIQUE INDEX "financecommission_active_source_employee" ON "finance_commissions"("verification_id","employee_id") WHERE "status"<>'CANCELLED';
CREATE INDEX "financecommission_source_employee_status" ON "finance_commissions"("verification_id","employee_id","status");
