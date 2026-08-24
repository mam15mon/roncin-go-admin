-- 保存费用创建时使用的汇率来源与日期快照，后续汇率设置变更不回写历史费用。
ALTER TABLE "order_fees"
  ALTER COLUMN "exchange_rate" TYPE numeric(18,8),
  ADD COLUMN "exchange_rate_source" character varying NOT NULL DEFAULT 'MANUAL',
  ADD COLUMN "exchange_rate_date" character varying(10),
  ADD COLUMN "exchange_rate_setting_id" uuid NULL;

UPDATE "order_fees" SET "exchange_rate_date" = "expense_date" WHERE "exchange_rate_date" IS NULL;

ALTER TABLE "order_fees"
  ALTER COLUMN "exchange_rate_source" DROP DEFAULT,
  ALTER COLUMN "exchange_rate_date" SET NOT NULL,
  ADD CONSTRAINT "order_fees_exchange_rate_source_check" CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'MANUAL')),
  ADD CONSTRAINT "order_fees_exchange_rate_settings_fees"
    FOREIGN KEY ("exchange_rate_setting_id") REFERENCES "exchange_rate_settings" ("id") ON DELETE NO ACTION;

CREATE INDEX "orderfee_exchange_rate_setting_id" ON "order_fees" ("exchange_rate_setting_id");
