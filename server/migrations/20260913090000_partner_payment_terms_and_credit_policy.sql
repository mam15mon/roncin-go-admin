ALTER TABLE "partner_settlement_rules"
  ADD COLUMN "payment_terms_days" bigint;

ALTER TABLE "finance_custom_settings"
  ADD COLUMN "credit_limit_selection_allowed" boolean NOT NULL DEFAULT true;
