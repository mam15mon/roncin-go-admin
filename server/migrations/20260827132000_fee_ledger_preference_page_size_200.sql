ALTER TABLE "finance_fee_ledger_preferences"
  DROP CONSTRAINT "finance_fee_ledger_preference_page_size_check";

ALTER TABLE "finance_fee_ledger_preferences"
  ADD CONSTRAINT "finance_fee_ledger_preference_page_size_check"
  CHECK ("page_size" IN (40, 60, 100, 200));
