-- 财务汇率闭环：按业务节点固化汇率快照，并在核销时确认汇兑损益。

ALTER TABLE "exchange_rate_time_standards"
  DROP CONSTRAINT IF EXISTS "exchange_rate_time_standards_time_standard_check";

DELETE FROM "exchange_rate_time_standards"
WHERE "rate_type" IN ('INVOICE', 'SETTLEMENT', 'BILL');

ALTER TABLE "exchange_rate_time_standards"
  ADD CONSTRAINT "exchange_rate_time_standards_time_standard_check"
  CHECK ("time_standard" IN (
    'ETD_ETA_TRAIN_DATE', 'BUSINESS_TIME', 'BARGE_ETD', 'EXPENSE_TIME',
    'ORDER_CREATED_AT', 'BILL_DATE', 'INVOICE_DATE', 'TRANSACTION_DATE', 'WRITE_OFF_TIME'
  ));

WITH headquarters AS (
  SELECT "id" FROM "organizations"
  WHERE "kind" = 'headquarters' AND "parent_id" IS NULL
), defaults("rate_type", "time_standard") AS (
  VALUES
    ('INVOICE', 'INVOICE_DATE'),
    ('SETTLEMENT', 'TRANSACTION_DATE'),
    ('BILL', 'BILL_DATE')
)
INSERT INTO "exchange_rate_time_standards" (
  "id", "created_at", "updated_at", "organization_id", "rate_type", "time_standard", "sort_order"
)
SELECT
  md5(headquarters."id"::text || ':' || defaults."rate_type" || ':' || defaults."time_standard")::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  headquarters."id",
  defaults."rate_type",
  defaults."time_standard",
  0
FROM headquarters CROSS JOIN defaults;

-- 新业务汇率类型此前尚未进入业务用例。以现有折本币汇率作为初始快照复制，
-- 后续可按开票、结算、账单和核销口径分别维护，不让升级后的外币流程立即中断。
INSERT INTO "exchange_rate_settings" (
  "id", "created_at", "updated_at", "organization_id", "rate_type",
  "from_currency", "to_currency", "effective_from", "effective_to",
  "receivable_rate", "payable_rate", "is_active"
)
SELECT
  md5(source."id"::text || ':' || target."rate_type")::uuid,
  source."created_at",
  CURRENT_TIMESTAMP,
  source."organization_id",
  target."rate_type",
  source."from_currency",
  source."to_currency",
  source."effective_from",
  source."effective_to",
  source."receivable_rate",
  source."payable_rate",
  source."is_active"
FROM "exchange_rate_settings" AS source
CROSS JOIN (VALUES ('INVOICE'), ('SETTLEMENT'), ('WRITE_OFF'), ('BILL')) AS target("rate_type")
WHERE source."rate_type" = 'BASE_CURRENCY'
ON CONFLICT ("organization_id", "rate_type", "from_currency", "to_currency", "effective_from") DO NOTHING;

ALTER TABLE "finance_bills"
  ADD COLUMN "exchange_rate" numeric(18,8),
  ADD COLUMN "exchange_rate_source" character varying,
  ADD COLUMN "exchange_rate_date" character varying(10),
  ADD COLUMN "exchange_rate_setting_id" uuid;

UPDATE "finance_bills"
SET
  "exchange_rate" = ROUND("base_currency_amount" / "total_amount", 8),
  "exchange_rate_source" = 'DERIVED',
  "exchange_rate_date" = "bill_date";

ALTER TABLE "finance_bills"
  ALTER COLUMN "exchange_rate" SET NOT NULL,
  ALTER COLUMN "exchange_rate_source" SET NOT NULL,
  ALTER COLUMN "exchange_rate_date" SET NOT NULL,
  ADD CONSTRAINT "finance_bills_exchange_rate_positive" CHECK ("exchange_rate" > 0),
  ADD CONSTRAINT "finance_bills_exchange_rate_source_check" CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'MANUAL', 'DERIVED'));

ALTER TABLE "finance_invoices"
  ADD COLUMN "base_currency" character varying(3),
  ADD COLUMN "exchange_rate" numeric(18,8),
  ADD COLUMN "exchange_rate_source" character varying,
  ADD COLUMN "exchange_rate_date" character varying(10),
  ADD COLUMN "exchange_rate_setting_id" uuid,
  ADD COLUMN "base_currency_amount" numeric(28,8);

UPDATE "finance_invoices" AS invoice
SET "base_currency" = source."base_currency"
FROM (
  SELECT link."invoice_id", MIN(bill."base_currency") AS "base_currency"
  FROM "finance_invoice_bills" AS link
  JOIN "finance_bills" AS bill ON bill."id" = link."bill_id"
  GROUP BY link."invoice_id"
) AS source
WHERE source."invoice_id" = invoice."id";

UPDATE "finance_invoices" AS invoice
SET
  "base_currency_amount" = source."base_currency_amount",
  "exchange_rate" = ROUND(source."base_currency_amount" / invoice."total_amount", 8),
  "exchange_rate_source" = 'DERIVED',
  "exchange_rate_date" = invoice."invoice_date"
FROM (
  SELECT link."invoice_id", SUM(bill."base_currency_amount") AS "base_currency_amount"
  FROM "finance_invoice_bills" AS link
  JOIN "finance_bills" AS bill ON bill."id" = link."bill_id"
  WHERE link."active" = true
  GROUP BY link."invoice_id"
) AS source
WHERE source."invoice_id" = invoice."id" AND invoice."invoice_date" IS NOT NULL;

ALTER TABLE "finance_invoices"
  ALTER COLUMN "base_currency" SET NOT NULL,
  ADD CONSTRAINT "finance_invoices_exchange_rate_positive" CHECK ("exchange_rate" IS NULL OR "exchange_rate" > 0),
  ADD CONSTRAINT "finance_invoices_exchange_rate_source_check" CHECK ("exchange_rate_source" IS NULL OR "exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'MANUAL', 'DERIVED')),
  ADD CONSTRAINT "finance_invoices_exchange_snapshot_consistency" CHECK (
    ("exchange_rate" IS NULL AND "exchange_rate_source" IS NULL AND "exchange_rate_date" IS NULL AND "base_currency_amount" IS NULL)
    OR
    ("exchange_rate" IS NOT NULL AND "exchange_rate_source" IS NOT NULL AND "exchange_rate_date" IS NOT NULL AND "base_currency_amount" IS NOT NULL)
  );

ALTER TABLE "finance_cashflows"
  ADD COLUMN "exchange_rate_source" character varying,
  ADD COLUMN "exchange_rate_date" character varying(10),
  ADD COLUMN "exchange_rate_setting_id" uuid;

UPDATE "finance_cashflows"
SET "exchange_rate_source" = 'MANUAL', "exchange_rate_date" = "transaction_date";

ALTER TABLE "finance_cashflows"
  ALTER COLUMN "exchange_rate_source" SET NOT NULL,
  ALTER COLUMN "exchange_rate_date" SET NOT NULL,
  ADD CONSTRAINT "finance_cashflows_exchange_rate_source_check" CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'MANUAL', 'DERIVED'));

ALTER TABLE "finance_verification_allocations"
  ADD COLUMN "bill_base_amount" numeric(28,8),
  ADD COLUMN "cashflow_base_amount" numeric(28,8),
  ADD COLUMN "write_off_base_amount" numeric(28,8),
  ADD COLUMN "exchange_gain_loss" numeric(28,8);

UPDATE "finance_verification_allocations" AS allocation
SET
  "bill_base_amount" = ROUND(bill."base_currency_amount" * allocation."amount" / bill."total_amount", 8),
  "cashflow_base_amount" = ROUND(cashflow."base_amount" * allocation."amount" / cashflow."amount", 8),
  "write_off_base_amount" = ROUND(cashflow."base_amount" * allocation."amount" / cashflow."amount", 8),
  "exchange_gain_loss" = CASE
    WHEN bill."direction" = 'RECEIVABLE' THEN
      ROUND(cashflow."base_amount" * allocation."amount" / cashflow."amount", 8)
      - ROUND(bill."base_currency_amount" * allocation."amount" / bill."total_amount", 8)
    ELSE
      ROUND(bill."base_currency_amount" * allocation."amount" / bill."total_amount", 8)
      - ROUND(cashflow."base_amount" * allocation."amount" / cashflow."amount", 8)
  END
FROM "finance_cashflows" AS cashflow, "finance_bills" AS bill
WHERE cashflow."id" = allocation."cashflow_id" AND bill."id" = allocation."bill_id";

ALTER TABLE "finance_verification_allocations"
  ALTER COLUMN "bill_base_amount" SET NOT NULL,
  ALTER COLUMN "cashflow_base_amount" SET NOT NULL,
  ALTER COLUMN "write_off_base_amount" SET NOT NULL,
  ALTER COLUMN "exchange_gain_loss" SET NOT NULL;

ALTER TABLE "finance_verifications"
  ADD COLUMN "base_currency" character varying(3),
  ADD COLUMN "exchange_rate" numeric(18,8),
  ADD COLUMN "exchange_rate_source" character varying,
  ADD COLUMN "exchange_rate_date" character varying(10),
  ADD COLUMN "exchange_rate_setting_id" uuid,
  ADD COLUMN "base_amount" numeric(28,8),
  ADD COLUMN "bill_base_amount" numeric(28,8),
  ADD COLUMN "cashflow_base_amount" numeric(28,8),
  ADD COLUMN "exchange_gain_loss" numeric(28,8);

UPDATE "finance_verifications" AS verification
SET
  "base_currency" = source."base_currency",
  "exchange_rate" = ROUND(source."write_off_base_amount" / verification."amount", 8),
  "exchange_rate_source" = 'DERIVED',
  "exchange_rate_date" = verification."verification_date",
  "base_amount" = source."write_off_base_amount",
  "bill_base_amount" = source."bill_base_amount",
  "cashflow_base_amount" = source."cashflow_base_amount",
  "exchange_gain_loss" = source."exchange_gain_loss"
FROM (
  SELECT
    allocation."verification_id",
    MIN(cashflow."base_currency") AS "base_currency",
    SUM(allocation."write_off_base_amount") AS "write_off_base_amount",
    SUM(allocation."bill_base_amount") AS "bill_base_amount",
    SUM(allocation."cashflow_base_amount") AS "cashflow_base_amount",
    SUM(allocation."exchange_gain_loss") AS "exchange_gain_loss"
  FROM "finance_verification_allocations" AS allocation
  JOIN "finance_cashflows" AS cashflow ON cashflow."id" = allocation."cashflow_id"
  GROUP BY allocation."verification_id"
) AS source
WHERE source."verification_id" = verification."id";

ALTER TABLE "finance_verifications"
  ALTER COLUMN "base_currency" SET NOT NULL,
  ALTER COLUMN "exchange_rate" SET NOT NULL,
  ALTER COLUMN "exchange_rate_source" SET NOT NULL,
  ALTER COLUMN "exchange_rate_date" SET NOT NULL,
  ALTER COLUMN "base_amount" SET NOT NULL,
  ALTER COLUMN "bill_base_amount" SET NOT NULL,
  ALTER COLUMN "cashflow_base_amount" SET NOT NULL,
  ALTER COLUMN "exchange_gain_loss" SET NOT NULL,
  ADD CONSTRAINT "finance_verifications_exchange_rate_positive" CHECK ("exchange_rate" > 0),
  ADD CONSTRAINT "finance_verifications_exchange_rate_source_check" CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'MANUAL', 'DERIVED'));
