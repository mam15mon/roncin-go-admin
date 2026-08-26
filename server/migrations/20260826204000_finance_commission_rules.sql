CREATE TABLE "finance_commission_rules"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "name" varchar(100) NOT NULL,
  "personnel_role" varchar NOT NULL,
  "calculation_basis" varchar NOT NULL,
  "rate_percent" numeric(7,4) NOT NULL,
  "effective_from" varchar(10),
  "effective_to" varchar(10),
  "enabled" boolean NOT NULL DEFAULT true,
  "note" varchar(500),
  "version" bigint NOT NULL DEFAULT 1,
  CONSTRAINT "commission_rule_role_check" CHECK("personnel_role" IN('SALES','OPERATOR')),
  CONSTRAINT "commission_rule_basis_check" CHECK("calculation_basis" IN('REALIZED_PROFIT','REALIZED_REVENUE')),
  CONSTRAINT "commission_rule_rate_check" CHECK("rate_percent">0 AND "rate_percent"<=100),
  CONSTRAINT "commission_rule_dates_check" CHECK("effective_from" IS NULL OR "effective_to" IS NULL OR "effective_from"<="effective_to")
);
CREATE UNIQUE INDEX "financecommissionrule_org_name" ON "finance_commission_rules"("organization_id","name");
CREATE INDEX "financecommissionrule_org_enabled_role" ON "finance_commission_rules"("organization_id","enabled","personnel_role");

ALTER TABLE "finance_commissions"
  ADD COLUMN "rule_id" uuid REFERENCES "finance_commission_rules"("id"),
  ADD COLUMN "rule_name" varchar(100),
  ADD COLUMN "personnel_role" varchar(20),
  ADD COLUMN "calculation_basis" varchar(30),
  ADD CONSTRAINT "finance_commission_rule_snapshot_check" CHECK(
    ("rule_id" IS NULL AND "rule_name" IS NULL AND "personnel_role" IS NULL AND "calculation_basis" IS NULL)
    OR ("rule_id" IS NOT NULL AND "rule_name" IS NOT NULL AND "personnel_role" IN('SALES','OPERATOR') AND "calculation_basis" IN('REALIZED_PROFIT','REALIZED_REVENUE'))
  );
DROP INDEX "financecommission_active_source_employee";
CREATE UNIQUE INDEX "financecommission_active_source_employee_rule" ON "finance_commissions"("verification_id","employee_id","rule_id") WHERE "status"<>'CANCELLED' AND "rule_id" IS NOT NULL;
CREATE INDEX "financecommission_rule" ON "finance_commissions"("rule_id","status");
