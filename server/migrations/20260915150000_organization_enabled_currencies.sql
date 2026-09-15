-- 组织支持的业务结算币种列表（JSONB 数组，例如 '["CNY", "USD", "EUR"]'）
-- 为 NULL 时默认继承组织本位币与核心外币主流子集（USD, GBP, EUR, CHF, AUD, SGD, HKD, THB, CNY）。
ALTER TABLE "organizations" ADD COLUMN "enabled_currencies" jsonb NULL;
