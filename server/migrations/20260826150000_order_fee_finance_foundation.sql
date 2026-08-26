-- 补齐费用进入账单前所需的幂等、状态、税额、本币金额和乐观锁快照。
ALTER TABLE "order_fees"
  ADD COLUMN "idempotency_key" character varying(128),
  ADD COLUMN "status" character varying NOT NULL DEFAULT 'DRAFT',
  ADD COLUMN "tax_inclusive" boolean NOT NULL DEFAULT true,
  ADD COLUMN "net_amount" numeric(28,8),
  ADD COLUMN "tax_amount" numeric(28,8),
  ADD COLUMN "base_currency" character varying(3),
  ADD COLUMN "base_currency_amount" numeric(28,8),
  ADD COLUMN "version" bigint NOT NULL DEFAULT 1,
  ADD COLUMN "cancelled_at" timestamp with time zone NULL,
  ADD COLUMN "cancelled_by" uuid NULL,
  ADD COLUMN "cancellation_reason" character varying(500) NULL;

WITH RECURSIVE organization_currency AS (
  SELECT
    organization."id" AS "origin_id",
    organization."id",
    organization."parent_id",
    organization."base_currency",
    0 AS "depth"
  FROM "organizations" AS organization

  UNION ALL

  SELECT
    child."origin_id",
    parent."id",
    parent."parent_id",
    parent."base_currency",
    child."depth" + 1
  FROM organization_currency AS child
  JOIN "organizations" AS parent ON parent."id" = child."parent_id"
  WHERE child."base_currency" IS NULL
), resolved_currency AS (
  SELECT DISTINCT ON ("origin_id")
    "origin_id",
    "base_currency"
  FROM organization_currency
  WHERE "base_currency" IS NOT NULL
  ORDER BY "origin_id", "depth"
)
UPDATE "order_fees" AS fee
SET
  "idempotency_key" = 'legacy:' || fee."id"::text,
  "net_amount" = round(
    fee."total_amount" / (1 + COALESCE(fee."tax_rate", 0) / 100),
    8
  ),
  "tax_amount" = fee."total_amount" - round(
    fee."total_amount" / (1 + COALESCE(fee."tax_rate", 0) / 100),
    8
  ),
  "base_currency" = resolved_currency."base_currency",
  "base_currency_amount" = round(fee."total_amount" * fee."exchange_rate", 8)
FROM "orders" AS business_order
JOIN resolved_currency ON resolved_currency."origin_id" = business_order."organization_id"
WHERE business_order."id" = fee."order_id";

ALTER TABLE "order_fees"
  ALTER COLUMN "idempotency_key" SET NOT NULL,
  ALTER COLUMN "net_amount" SET NOT NULL,
  ALTER COLUMN "tax_amount" SET NOT NULL,
  ALTER COLUMN "base_currency" SET NOT NULL,
  ALTER COLUMN "base_currency_amount" SET NOT NULL,
  ADD CONSTRAINT "order_fees_status_check"
    CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'BILLED', 'CANCELLED')),
  ADD CONSTRAINT "order_fees_net_amount_nonnegative" CHECK ("net_amount" >= 0),
  ADD CONSTRAINT "order_fees_tax_amount_nonnegative" CHECK ("tax_amount" >= 0),
  ADD CONSTRAINT "order_fees_base_currency_amount_positive" CHECK ("base_currency_amount" > 0),
  ADD CONSTRAINT "order_fees_version_positive" CHECK ("version" > 0),
  ADD CONSTRAINT "order_fees_users_cancelled_by"
    FOREIGN KEY ("cancelled_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "order_fees_cancellation_consistency"
    CHECK (
      ("status" = 'CANCELLED' AND "cancelled_at" IS NOT NULL AND "cancelled_by" IS NOT NULL AND "cancellation_reason" IS NOT NULL)
      OR
      ("status" <> 'CANCELLED' AND "cancelled_at" IS NULL AND "cancelled_by" IS NULL AND "cancellation_reason" IS NULL)
    );

CREATE UNIQUE INDEX "orderfee_order_id_idempotency_key"
  ON "order_fees" ("order_id", "idempotency_key");
CREATE INDEX "orderfee_order_id_status_created_at"
  ON "order_fees" ("order_id", "status", "created_at");
