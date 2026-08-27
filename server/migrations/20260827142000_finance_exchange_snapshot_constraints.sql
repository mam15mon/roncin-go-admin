-- 汇率快照引用与金额组成由数据库兜底，避免绕过领域层写入失真的财务数据。
ALTER TABLE "finance_bills"
  ADD CONSTRAINT "finance_bills_exchange_rate_setting_fk"
    FOREIGN KEY ("exchange_rate_setting_id") REFERENCES "exchange_rate_settings" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "finance_bills_base_amount_composition"
    CHECK ("exchange_rate_source" = 'DERIVED' OR "base_currency_amount" = ROUND("total_amount" * "exchange_rate", 8));
CREATE INDEX "finance_bills_exchange_rate_setting_id"
  ON "finance_bills" ("exchange_rate_setting_id");

ALTER TABLE "finance_invoices"
  ADD CONSTRAINT "finance_invoices_exchange_rate_setting_fk"
    FOREIGN KEY ("exchange_rate_setting_id") REFERENCES "exchange_rate_settings" ("id") ON DELETE NO ACTION,
  DROP CONSTRAINT "finance_invoices_exchange_snapshot_consistency",
  ADD CONSTRAINT "finance_invoices_exchange_snapshot_consistency" CHECK (
    (
      "status" = 'DRAFT'
      AND "exchange_rate" IS NULL
      AND "exchange_rate_source" IS NULL
      AND "exchange_rate_date" IS NULL
      AND "base_currency_amount" IS NULL
    )
    OR (
      "status" IN ('ISSUED', 'RED_FLUSHED')
      AND "exchange_rate" IS NOT NULL
      AND "exchange_rate_source" IS NOT NULL
      AND "exchange_rate_date" IS NOT NULL
      AND ("exchange_rate_source" = 'DERIVED' OR "base_currency_amount" = ROUND("total_amount" * "exchange_rate", 8))
    )
    OR (
      "status" = 'CANCELLED'
      AND (
        ("exchange_rate" IS NULL AND "exchange_rate_source" IS NULL AND "exchange_rate_date" IS NULL AND "base_currency_amount" IS NULL)
        OR
        ("exchange_rate" IS NOT NULL AND "exchange_rate_source" IS NOT NULL AND "exchange_rate_date" IS NOT NULL AND ("exchange_rate_source" = 'DERIVED' OR "base_currency_amount" = ROUND("total_amount" * "exchange_rate", 8)))
      )
    )
  );
CREATE INDEX "finance_invoices_exchange_rate_setting_id"
  ON "finance_invoices" ("exchange_rate_setting_id");

ALTER TABLE "finance_cashflows"
  ADD CONSTRAINT "finance_cashflows_exchange_rate_setting_fk"
    FOREIGN KEY ("exchange_rate_setting_id") REFERENCES "exchange_rate_settings" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "finance_cashflows_base_amount_composition"
    CHECK ("base_amount" = ROUND("amount" * "exchange_rate", 8));
CREATE INDEX "finance_cashflows_exchange_rate_setting_id"
  ON "finance_cashflows" ("exchange_rate_setting_id");

ALTER TABLE "finance_verifications"
  ADD CONSTRAINT "finance_verifications_exchange_rate_setting_fk"
    FOREIGN KEY ("exchange_rate_setting_id") REFERENCES "exchange_rate_settings" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "finance_verifications_base_amount_composition"
    CHECK ("exchange_rate_source" = 'DERIVED' OR "base_amount" = ROUND("amount" * "exchange_rate", 8));
CREATE INDEX "finance_verifications_exchange_rate_setting_id"
  ON "finance_verifications" ("exchange_rate_setting_id");
