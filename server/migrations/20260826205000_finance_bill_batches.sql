CREATE TABLE "finance_bill_batches"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "batch_no" varchar(64) NOT NULL,
  "idempotency_key" varchar(128) NOT NULL,
  "request_hash" varchar(64) NOT NULL,
  "split_by_order" boolean NOT NULL DEFAULT false,
  "split_by_tax_rate" boolean NOT NULL DEFAULT false,
  "fee_count" bigint NOT NULL,
  "bill_count" bigint NOT NULL,
  "total_base_amount" numeric(28,8) NOT NULL,
  "base_currency" varchar(3) NOT NULL,
  "created_by" uuid NOT NULL REFERENCES "users"("id"),
  CONSTRAINT "finance_bill_batch_counts_check" CHECK("fee_count">0 AND "bill_count">0),
  CONSTRAINT "finance_bill_batch_request_hash_check" CHECK(length("request_hash")=64)
);
CREATE UNIQUE INDEX "financebillbatch_organization_id_batch_no" ON "finance_bill_batches"("organization_id","batch_no");
CREATE UNIQUE INDEX "financebillbatch_organization_id_idempotency_key" ON "finance_bill_batches"("organization_id","idempotency_key");
CREATE INDEX "financebillbatch_organization_id_created_at" ON "finance_bill_batches"("organization_id","created_at");

ALTER TABLE "finance_bills"
  ADD COLUMN "batch_id" uuid REFERENCES "finance_bill_batches"("id"),
  ADD COLUMN "statement_title" varchar(200),
  ADD COLUMN "payment_terms_days" bigint,
  ADD CONSTRAINT "finance_bill_payment_terms_check" CHECK("payment_terms_days" IS NULL OR "payment_terms_days" BETWEEN 0 AND 3650);
CREATE INDEX "financebill_batch_id" ON "finance_bills"("batch_id");

ALTER TABLE "finance_bill_lines"
  ADD COLUMN "tax_rate" numeric(7,4),
  ADD CONSTRAINT "finance_bill_line_tax_rate_check" CHECK("tax_rate" IS NULL OR "tax_rate" BETWEEN 0 AND 100);

INSERT INTO "number_rules"("id","created_at","updated_at","document_type","prefix","date_format","sequence_length","reset_policy","enabled","organization_id")
SELECT gen_random_uuid(),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'bill_batch','BG','yyyyMMdd',5,'daily',TRUE,"id"
FROM "organizations";
