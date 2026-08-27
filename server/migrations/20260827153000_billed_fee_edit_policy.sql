-- 账单创建后的费用修改策略。默认关闭，升级后不改变既有财务锁定口径。
CREATE TABLE "finance_custom_settings" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL,
  "billed_fee_edit_enabled" boolean NOT NULL DEFAULT false,
  "billed_fee_name_editable" boolean NOT NULL DEFAULT false,
  "billed_fee_currency_editable" boolean NOT NULL DEFAULT false,
  "billed_fee_exchange_rate_editable" boolean NOT NULL DEFAULT false,
  "billed_fee_quantity_editable" boolean NOT NULL DEFAULT false,
  "billed_fee_unit_price_editable" boolean NOT NULL DEFAULT false,
  "billed_fee_tax_rate_editable" boolean NOT NULL DEFAULT false,
  "version" bigint NOT NULL DEFAULT 1,
  "updated_by" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_custom_settings_version_positive" CHECK ("version" > 0),
  CONSTRAINT "finance_custom_settings_organization_fk"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_custom_settings_updated_by_fk"
    FOREIGN KEY ("updated_by") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "finance_custom_setting_organization_unique"
  ON "finance_custom_settings" ("organization_id");
CREATE INDEX "finance_custom_settings_updated_at"
  ON "finance_custom_settings" ("updated_at");

-- 账单行保存修改后的数量与单价快照；历史行按原费用回填。
ALTER TABLE "finance_bill_lines"
  ADD COLUMN "quantity" numeric(18,4),
  ADD COLUMN "unit_price" numeric(18,4);

UPDATE "finance_bill_lines" AS line
SET "quantity" = fee."quantity", "unit_price" = fee."unit_price"
FROM "order_fees" AS fee
WHERE fee."id" = line."order_fee_id";

ALTER TABLE "finance_bill_lines"
  ALTER COLUMN "quantity" SET NOT NULL,
  ALTER COLUMN "unit_price" SET NOT NULL,
  ADD CONSTRAINT "finance_bill_lines_quantity_positive" CHECK ("quantity" > 0),
  ADD CONSTRAINT "finance_bill_lines_unit_price_positive" CHECK ("unit_price" > 0);
