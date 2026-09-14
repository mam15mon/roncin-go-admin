-- 汇率实战化内核（阶段二：周汇率 + 买卖点差 + 中国银行一键同步）：
-- 1) exchange_rate_settings 新增 source 列（行写入来源 MANUAL/IMPORT/BOC_SYNC），
--    费用快照来源据此区分 WEEKLY 与 BOC_SYNC；
-- 2) 快照来源枚举退役值清理：BASE_CURRENCY / INHERITED_BASE_CURRENCY 随
--    ResolveBaseRate 退役一并清理（历史值先重映射为 SYSTEM / DERIVED）。
--    Ent 枚举在 PostgreSQL 中为 varchar，枚举集变化无 DDL 列类型变更，
--    但需重映射存量值并同步列 COMMENT（如有）。

-- 1. 行写入来源列（默认 MANUAL：历史行均视为手工维护口径）。
ALTER TABLE "exchange_rate_settings"
  ADD COLUMN IF NOT EXISTS "source" varchar NOT NULL DEFAULT 'MANUAL';

-- 2. 快照来源枚举退役值清理（开发数据处置已获用户授权，prd.md §8.5；
--    重映射保证任何存量库态迁移后枚举值全部落在新生效集内）。
UPDATE "order_fees" SET "exchange_rate_source" = 'SYSTEM'
  WHERE "exchange_rate_source" = 'BASE_CURRENCY';

UPDATE "finance_bills" SET "exchange_rate_source" = 'DERIVED'
  WHERE "exchange_rate_source" = 'INHERITED_BASE_CURRENCY';

UPDATE "finance_bills" SET "exchange_rate_source" = 'SYSTEM'
  WHERE "exchange_rate_source" = 'BASE_CURRENCY';

UPDATE "finance_invoices" SET "exchange_rate_source" = 'DERIVED'
  WHERE "exchange_rate_source" = 'INHERITED_BASE_CURRENCY';

UPDATE "finance_invoices" SET "exchange_rate_source" = 'SYSTEM'
  WHERE "exchange_rate_source" = 'BASE_CURRENCY';

UPDATE "finance_cashflows" SET "exchange_rate_source" = 'DERIVED'
  WHERE "exchange_rate_source" = 'INHERITED_BASE_CURRENCY';

UPDATE "finance_cashflows" SET "exchange_rate_source" = 'SYSTEM'
  WHERE "exchange_rate_source" = 'BASE_CURRENCY';

-- 2.1 快照来源 CHECK 约束重建为新枚举集：数据重映射后旧约束仍只认退役值集，
--     不重建会使 WEEKLY / INHERITED_LAST_WEEK / BOC_SYNC 写入直接 23514
--     （constraint 名与列集沿承既有迁移链，DROP 带 IF EXISTS 容忍建库漂移）。
ALTER TABLE "order_fees"
  DROP CONSTRAINT IF EXISTS "order_fees_exchange_rate_source_check",
  ADD CONSTRAINT "order_fees_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'MANUAL', 'DERIVED', 'WEEKLY', 'INHERITED_LAST_WEEK', 'BOC_SYNC'));

ALTER TABLE "finance_bills"
  DROP CONSTRAINT IF EXISTS "finance_bills_exchange_rate_source_check",
  ADD CONSTRAINT "finance_bills_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'MANUAL', 'DERIVED', 'WEEKLY', 'INHERITED_LAST_WEEK', 'BOC_SYNC'));

ALTER TABLE "finance_invoices"
  DROP CONSTRAINT IF EXISTS "finance_invoices_exchange_rate_source_check",
  ADD CONSTRAINT "finance_invoices_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IS NULL OR "exchange_rate_source" IN ('SYSTEM', 'MANUAL', 'DERIVED', 'WEEKLY', 'INHERITED_LAST_WEEK', 'BOC_SYNC'));

ALTER TABLE "finance_cashflows"
  DROP CONSTRAINT IF EXISTS "finance_cashflows_exchange_rate_source_check",
  ADD CONSTRAINT "finance_cashflows_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'MANUAL', 'DERIVED', 'WEEKLY', 'INHERITED_LAST_WEEK', 'BOC_SYNC'));

-- 3. notification_deliveries.template 新增 EXCHANGE_RATE_WEEKLY_REMINDER
--    （Ent 枚举为 varchar，无 DDL 变更；新值由督办 worker 写入）。
