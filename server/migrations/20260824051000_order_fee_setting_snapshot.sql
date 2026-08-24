-- 订单费用保存费用设置引用和录入时快照；历史费用的新增字段保持为空。
ALTER TABLE "order_fees"
  ADD COLUMN "fee_setting_id" uuid NULL,
  ADD COLUMN "billing_unit_id" uuid NULL,
  ADD COLUMN "fee_name_en" character varying(128) NULL,
  ADD COLUMN "tax_rate" numeric(5,2) NULL,
  ADD COLUMN "taxable_service_name" character varying(128) NULL,
  ADD CONSTRAINT "order_fees_fee_settings_order_fees"
    FOREIGN KEY ("fee_setting_id") REFERENCES "fee_settings" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "order_fees_billing_units_order_fees"
    FOREIGN KEY ("billing_unit_id") REFERENCES "billing_units" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "order_fees_tax_rate_range"
    CHECK ("tax_rate" IS NULL OR ("tax_rate" >= 0 AND "tax_rate" <= 100));

CREATE INDEX "orderfee_fee_setting_id" ON "order_fees" ("fee_setting_id");
CREATE INDEX "orderfee_billing_unit_id" ON "order_fees" ("billing_unit_id");
