-- 账单聚合层：账单头保存结算口径，账单行保存入账时费用快照。
CREATE TABLE "finance_bills" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "bill_no" character varying(64) NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "direction" character varying NOT NULL,
  "status" character varying NOT NULL DEFAULT 'DRAFT',
  "settlement_party_id" uuid NOT NULL,
  "settlement_party_name" character varying(200) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "base_currency" character varying(3) NOT NULL,
  "total_amount" numeric(28,8) NOT NULL,
  "net_amount" numeric(28,8) NOT NULL,
  "tax_amount" numeric(28,8) NOT NULL,
  "base_currency_amount" numeric(28,8) NOT NULL,
  "fee_count" bigint NOT NULL,
  "bill_date" character varying(10) NOT NULL,
  "due_date" character varying(10) NULL,
  "note" character varying(500) NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "confirmed_at" timestamp with time zone NULL,
  "confirmed_by" uuid NULL,
  "cancelled_at" timestamp with time zone NULL,
  "cancelled_by" uuid NULL,
  "cancellation_reason" character varying(500) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_bills_organizations_finance_bills"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bills_partners_finance_bills"
    FOREIGN KEY ("settlement_party_id") REFERENCES "partners" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bills_users_confirmed_finance_bills"
    FOREIGN KEY ("confirmed_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bills_users_cancelled_finance_bills"
    FOREIGN KEY ("cancelled_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bills_direction_check" CHECK ("direction" IN ('RECEIVABLE', 'PAYABLE')),
  CONSTRAINT "finance_bills_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'CANCELLED')),
  CONSTRAINT "finance_bills_amount_positive" CHECK ("total_amount" > 0 AND "base_currency_amount" > 0),
  CONSTRAINT "finance_bills_tax_nonnegative" CHECK ("net_amount" >= 0 AND "tax_amount" >= 0),
  CONSTRAINT "finance_bills_amount_composition" CHECK ("total_amount" = "net_amount" + "tax_amount"),
  CONSTRAINT "finance_bills_fee_count_positive" CHECK ("fee_count" > 0),
  CONSTRAINT "finance_bills_version_positive" CHECK ("version" > 0),
  CONSTRAINT "finance_bills_due_date_order" CHECK ("due_date" IS NULL OR "due_date" >= "bill_date"),
  CONSTRAINT "finance_bills_confirmation_consistency" CHECK (
    ("status" = 'DRAFT' AND "confirmed_at" IS NULL AND "confirmed_by" IS NULL)
    OR ("status" <> 'DRAFT' AND (("confirmed_at" IS NULL AND "confirmed_by" IS NULL) OR ("confirmed_at" IS NOT NULL AND "confirmed_by" IS NOT NULL)))
  ),
  CONSTRAINT "finance_bills_cancellation_consistency" CHECK (
    ("status" = 'CANCELLED' AND "cancelled_at" IS NOT NULL AND "cancelled_by" IS NOT NULL AND "cancellation_reason" IS NOT NULL)
    OR ("status" <> 'CANCELLED' AND "cancelled_at" IS NULL AND "cancelled_by" IS NULL AND "cancellation_reason" IS NULL)
  )
);

CREATE TABLE "finance_bill_lines" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "bill_id" uuid NOT NULL,
  "order_fee_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "order_no" character varying(64) NOT NULL,
  "fee_code" character varying(30) NOT NULL,
  "fee_name" character varying(80) NOT NULL,
  "total_amount" numeric(28,8) NOT NULL,
  "net_amount" numeric(28,8) NOT NULL,
  "tax_amount" numeric(28,8) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "exchange_rate" numeric(18,8) NOT NULL,
  "base_currency" character varying(3) NOT NULL,
  "base_currency_amount" numeric(28,8) NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_bill_lines_finance_bills_lines"
    FOREIGN KEY ("bill_id") REFERENCES "finance_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bill_lines_order_fees_finance_bill_lines"
    FOREIGN KEY ("order_fee_id") REFERENCES "order_fees" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bill_lines_orders_finance_bill_lines"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_bill_lines_amount_positive" CHECK ("total_amount" > 0 AND "base_currency_amount" > 0),
  CONSTRAINT "finance_bill_lines_tax_nonnegative" CHECK ("net_amount" >= 0 AND "tax_amount" >= 0),
  CONSTRAINT "finance_bill_lines_amount_composition" CHECK ("total_amount" = "net_amount" + "tax_amount")
);

CREATE UNIQUE INDEX "financebill_organization_id_bill_no"
  ON "finance_bills" ("organization_id", "bill_no");
CREATE UNIQUE INDEX "financebill_organization_id_idempotency_key"
  ON "finance_bills" ("organization_id", "idempotency_key");
CREATE INDEX "financebill_organization_id_status_bill_date"
  ON "finance_bills" ("organization_id", "status", "bill_date");
CREATE INDEX "financebill_settlement_party_id_direction_currency"
  ON "finance_bills" ("settlement_party_id", "direction", "currency");
CREATE INDEX "finance_bill_lines_bill_id_active"
  ON "finance_bill_lines" ("bill_id", "active");
CREATE INDEX "finance_bill_lines_order_fee_id_active"
  ON "finance_bill_lines" ("order_fee_id", "active");
CREATE INDEX "finance_bill_lines_order_id"
  ON "finance_bill_lines" ("order_id");
CREATE UNIQUE INDEX "finance_bill_lines_active_fee_unique"
  ON "finance_bill_lines" ("order_fee_id") WHERE "active" = true;
