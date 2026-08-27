-- 汇率自定义设置：专用汇率缺失时，可显式选择继承同期折本币汇率。
-- 默认不创建配置记录，等价于关闭；避免升级后静默改变既有财务口径。
CREATE TABLE "exchange_rate_custom_settings" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL,
  "inherit_base_currency_rate" boolean NOT NULL DEFAULT false,
  "version" bigint NOT NULL DEFAULT 1,
  "updated_by" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "exchange_rate_custom_settings_version_positive" CHECK ("version" > 0),
  CONSTRAINT "exchange_rate_custom_settings_organization_fk"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "exchange_rate_custom_settings_updated_by_fk"
    FOREIGN KEY ("updated_by") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "exchange_rate_custom_setting_organization_unique"
  ON "exchange_rate_custom_settings" ("organization_id");
CREATE INDEX "exchange_rate_custom_settings_updated_at"
  ON "exchange_rate_custom_settings" ("updated_at");

-- 继承结果需要进入各财务节点的不可变汇率快照，明确区别于专用系统汇率和本币 1:1。
ALTER TABLE "finance_bills"
  DROP CONSTRAINT "finance_bills_exchange_rate_source_check",
  ADD CONSTRAINT "finance_bills_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'INHERITED_BASE_CURRENCY', 'MANUAL', 'DERIVED'));

ALTER TABLE "finance_invoices"
  DROP CONSTRAINT "finance_invoices_exchange_rate_source_check",
  ADD CONSTRAINT "finance_invoices_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IS NULL OR "exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'INHERITED_BASE_CURRENCY', 'MANUAL', 'DERIVED'));

ALTER TABLE "finance_cashflows"
  DROP CONSTRAINT "finance_cashflows_exchange_rate_source_check",
  ADD CONSTRAINT "finance_cashflows_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'INHERITED_BASE_CURRENCY', 'MANUAL', 'DERIVED'));

ALTER TABLE "finance_verifications"
  DROP CONSTRAINT "finance_verifications_exchange_rate_source_check",
  ADD CONSTRAINT "finance_verifications_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'INHERITED_BASE_CURRENCY', 'MANUAL', 'DERIVED'));
