-- 税务开票层：内部开票记录与税务发票号码分离，账单关联保留历史且只允许一个有效关联。
UPDATE "number_rules"
SET "prefix" = 'IN', "enabled" = true, "updated_at" = CURRENT_TIMESTAMP
WHERE "document_type" = 'invoice' AND "enabled" = false;

CREATE TABLE "finance_invoices" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "record_no" character varying(64) NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "direction" character varying NOT NULL,
  "status" character varying NOT NULL DEFAULT 'DRAFT',
  "invoice_type" character varying NOT NULL,
  "settlement_party_id" uuid NOT NULL,
  "settlement_party_name" character varying(200) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "total_amount" numeric(28,8) NOT NULL,
  "tax_amount" numeric(28,8) NOT NULL,
  "bill_count" bigint NOT NULL,
  "tax_invoice_no" character varying(100) NULL,
  "invoice_date" character varying(10) NULL,
  "note" character varying(500) NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "issued_at" timestamp with time zone NULL,
  "issued_by" uuid NULL,
  "cancelled_at" timestamp with time zone NULL,
  "cancelled_by" uuid NULL,
  "cancellation_reason" character varying(500) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_invoices_organizations_finance_invoices" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_invoices_partners_finance_invoices" FOREIGN KEY ("settlement_party_id") REFERENCES "partners" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_invoices_users_issued_finance_invoices" FOREIGN KEY ("issued_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_invoices_users_cancelled_finance_invoices" FOREIGN KEY ("cancelled_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_invoices_direction_check" CHECK ("direction" IN ('RECEIVABLE', 'PAYABLE')),
  CONSTRAINT "finance_invoices_status_check" CHECK ("status" IN ('DRAFT', 'ISSUED', 'CANCELLED')),
  CONSTRAINT "finance_invoices_type_check" CHECK ("invoice_type" IN ('NORMAL', 'SPECIAL')),
  CONSTRAINT "finance_invoices_amount_positive" CHECK ("total_amount" > 0 AND "tax_amount" >= 0),
  CONSTRAINT "finance_invoices_bill_count_positive" CHECK ("bill_count" > 0),
  CONSTRAINT "finance_invoices_version_positive" CHECK ("version" > 0),
  CONSTRAINT "finance_invoices_issue_consistency" CHECK (
    ("status" = 'DRAFT' AND "tax_invoice_no" IS NULL AND "invoice_date" IS NULL AND "issued_at" IS NULL AND "issued_by" IS NULL)
    OR ("status" <> 'DRAFT' AND (("tax_invoice_no" IS NULL AND "invoice_date" IS NULL AND "issued_at" IS NULL AND "issued_by" IS NULL) OR ("tax_invoice_no" IS NOT NULL AND "invoice_date" IS NOT NULL AND "issued_at" IS NOT NULL AND "issued_by" IS NOT NULL)))
  ),
  CONSTRAINT "finance_invoices_cancel_consistency" CHECK (
    ("status" = 'CANCELLED' AND "cancelled_at" IS NOT NULL AND "cancelled_by" IS NOT NULL AND "cancellation_reason" IS NOT NULL)
    OR ("status" <> 'CANCELLED' AND "cancelled_at" IS NULL AND "cancelled_by" IS NULL AND "cancellation_reason" IS NULL)
  )
);

CREATE TABLE "finance_invoice_bills" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "invoice_id" uuid NOT NULL,
  "bill_id" uuid NOT NULL,
  "bill_no" character varying(64) NOT NULL,
  "amount" numeric(28,8) NOT NULL,
  "tax_amount" numeric(28,8) NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_invoice_bills_finance_invoices_bill_links" FOREIGN KEY ("invoice_id") REFERENCES "finance_invoices" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_invoice_bills_finance_bills_invoice_links" FOREIGN KEY ("bill_id") REFERENCES "finance_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_invoice_bills_amount_positive" CHECK ("amount" > 0 AND "tax_amount" >= 0)
);

CREATE UNIQUE INDEX "financeinvoice_organization_id_record_no" ON "finance_invoices" ("organization_id", "record_no");
CREATE UNIQUE INDEX "financeinvoice_organization_id_idempotency_key" ON "finance_invoices" ("organization_id", "idempotency_key");
CREATE INDEX "financeinvoice_organization_id_status_created_at" ON "finance_invoices" ("organization_id", "status", "created_at");
CREATE INDEX "financeinvoice_settlement_party_id_direction_currency" ON "finance_invoices" ("settlement_party_id", "direction", "currency");
CREATE INDEX "financeinvoice_tax_invoice_no" ON "finance_invoices" ("tax_invoice_no");
CREATE INDEX "financeinvoicebill_invoice_id_active" ON "finance_invoice_bills" ("invoice_id", "active");
CREATE INDEX "financeinvoicebill_bill_id_active" ON "finance_invoice_bills" ("bill_id", "active");
CREATE UNIQUE INDEX "finance_invoice_bills_active_bill_unique" ON "finance_invoice_bills" ("bill_id") WHERE "active" = true;
