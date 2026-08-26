ALTER TABLE "finance_commissions"
  ADD COLUMN "rule_version" bigint NOT NULL DEFAULT 1,
  ADD COLUMN "calculation_version" varchar(32) NOT NULL DEFAULT 'ORDER_LINE_V1',
  ADD COLUMN "source_fingerprint" varchar(64) NOT NULL DEFAULT '';

CREATE TABLE "finance_commission_lines"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "commission_id" uuid NOT NULL REFERENCES "finance_commissions"("id"),
  "order_id" uuid NOT NULL REFERENCES "orders"("id"),
  "order_no" varchar(64) NOT NULL,
  "personnel_assignment_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "employee_name" varchar(100) NOT NULL,
  "personnel_role" varchar(20) NOT NULL,
  "calculation_basis" varchar(30) NOT NULL,
  "base_currency" varchar(3) NOT NULL,
  "realized_revenue" numeric(28,8) NOT NULL,
  "allocated_cost" numeric(28,8) NOT NULL,
  "realized_profit" numeric(28,8) NOT NULL,
  "rate_percent" numeric(7,4) NOT NULL,
  "commission_amount" numeric(28,8) NOT NULL,
  CONSTRAINT "commission_line_role_check" CHECK("personnel_role" IN('SALES','OPERATOR')),
  CONSTRAINT "commission_line_basis_check" CHECK("calculation_basis" IN('REALIZED_PROFIT','REALIZED_REVENUE')),
  CONSTRAINT "commission_line_amount_nonnegative" CHECK("commission_amount">=0)
);

CREATE UNIQUE INDEX "financecommissionline_commission_order" ON "finance_commission_lines"("commission_id","order_id");
CREATE INDEX "financecommissionline_org_employee" ON "finance_commission_lines"("organization_id","employee_id");
CREATE INDEX "financecommissionline_order" ON "finance_commission_lines"("order_id");
