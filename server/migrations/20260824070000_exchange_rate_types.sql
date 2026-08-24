-- 汇率主数据支持折本币、开票、结算、核销和账单五种业务类型。
ALTER TABLE "exchange_rate_settings"
  DROP CONSTRAINT "exchange_rate_settings_rate_type_check";

ALTER TABLE "exchange_rate_settings"
  ADD CONSTRAINT "exchange_rate_settings_rate_type_check"
  CHECK ("rate_type" IN ('BASE_CURRENCY', 'INVOICE', 'SETTLEMENT', 'WRITE_OFF', 'BILL'));
