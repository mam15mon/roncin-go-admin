ALTER TABLE "finance_commissions"
  ADD COLUMN "adjustment_sequence" bigint NOT NULL DEFAULT 0;

CREATE TABLE "finance_commission_adjustments"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "commission_id" uuid NOT NULL REFERENCES "finance_commissions"("id"),
  "order_id" uuid NOT NULL REFERENCES "orders"("id"),
  "adjustment_no" varchar(80) NOT NULL,
  "idempotency_key" varchar(128) NOT NULL,
  "commission_no" varchar(64) NOT NULL,
  "order_no" varchar(64) NOT NULL,
  "employee_id" uuid NOT NULL REFERENCES "users"("id"),
  "employee_name" varchar(100) NOT NULL,
  "direction" varchar(16) NOT NULL,
  "status" varchar(16) NOT NULL DEFAULT 'DRAFT',
  "base_currency" varchar(3) NOT NULL,
  "amount" numeric(28,8) NOT NULL,
  "reason" varchar(500) NOT NULL,
  "note" varchar(500),
  "version" bigint NOT NULL DEFAULT 1,
  "confirmed_at" timestamptz,
  "confirmed_by" uuid REFERENCES "users"("id"),
  "paid_at" timestamptz,
  "paid_by" uuid REFERENCES "users"("id"),
  "cancelled_at" timestamptz,
  "cancelled_by" uuid REFERENCES "users"("id"),
  "cancellation_reason" varchar(500),
  CONSTRAINT "commission_adjustment_direction_check" CHECK("direction" IN('INCREASE','DECREASE')),
  CONSTRAINT "commission_adjustment_status_check" CHECK("status" IN('DRAFT','CONFIRMED','PAID','CANCELLED')),
  CONSTRAINT "commission_adjustment_amount_positive" CHECK("amount">0)
);

CREATE UNIQUE INDEX "financecommissionadjustment_org_no" ON "finance_commission_adjustments"("organization_id","adjustment_no");
CREATE UNIQUE INDEX "financecommissionadjustment_org_idempotency" ON "finance_commission_adjustments"("organization_id","idempotency_key");
CREATE INDEX "financecommissionadjustment_commission_status_created" ON "finance_commission_adjustments"("commission_id","status","created_at");
CREATE INDEX "financecommissionadjustment_order_status" ON "finance_commission_adjustments"("order_id","status");
