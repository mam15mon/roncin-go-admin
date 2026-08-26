CREATE TABLE "partner_invoice_profiles"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "partner_id" uuid NOT NULL REFERENCES "partners"("id"),
  "invoice_title" varchar(200) NOT NULL,
  "taxpayer_identification_no" varchar(64) NOT NULL,
  "registered_address" varchar(500) NOT NULL DEFAULT '',
  "registered_phone" varchar(50) NOT NULL DEFAULT '',
  "bank_name" varchar(200) NOT NULL DEFAULT '',
  "bank_account" varchar(100) NOT NULL DEFAULT '',
  "default_invoice_type" varchar NOT NULL DEFAULT 'NORMAL',
  "version" bigint NOT NULL DEFAULT 1,
  CONSTRAINT "partner_invoice_profile_type_check" CHECK("default_invoice_type" IN ('NORMAL','SPECIAL')),
  CONSTRAINT "partner_invoice_profile_version_check" CHECK("version">0)
);
CREATE UNIQUE INDEX "partnerinvoiceprofile_organization_id_partner_id" ON "partner_invoice_profiles"("organization_id","partner_id");
CREATE INDEX "partnerinvoiceprofile_organization_id_taxpayer_identification_no" ON "partner_invoice_profiles"("organization_id","taxpayer_identification_no");

ALTER TABLE "finance_invoices"
  ADD COLUMN "invoice_profile_id" uuid REFERENCES "partner_invoice_profiles"("id"),
  ADD COLUMN "invoice_title" varchar(200),
  ADD COLUMN "taxpayer_identification_no" varchar(64),
  ADD COLUMN "registered_address" varchar(500),
  ADD COLUMN "registered_phone" varchar(50),
  ADD COLUMN "bank_name" varchar(200),
  ADD COLUMN "bank_account" varchar(100),
  ADD COLUMN "net_amount" numeric(28,8);
UPDATE "finance_invoices" SET "net_amount"="total_amount"-"tax_amount";
ALTER TABLE "finance_invoices" ALTER COLUMN "net_amount" SET NOT NULL;
CREATE INDEX "financeinvoice_invoice_profile_id" ON "finance_invoices"("invoice_profile_id");

CREATE TABLE "finance_invoice_lines"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "invoice_id" uuid NOT NULL REFERENCES "finance_invoices"("id"),
  "line_no" bigint NOT NULL,
  "item_code" varchar(30) NOT NULL,
  "item_name" varchar(80) NOT NULL,
  "tax_rate" numeric(7,4) NOT NULL,
  "net_amount" numeric(28,8) NOT NULL,
  "tax_amount" numeric(28,8) NOT NULL,
  "total_amount" numeric(28,8) NOT NULL,
  "currency" varchar(3) NOT NULL,
  "source_line_count" bigint NOT NULL,
  CONSTRAINT "finance_invoice_line_values_check" CHECK("line_no">0 AND "source_line_count">0 AND "tax_rate" BETWEEN 0 AND 100)
);
CREATE UNIQUE INDEX "financeinvoiceline_invoice_id_line_no" ON "finance_invoice_lines"("invoice_id","line_no");
