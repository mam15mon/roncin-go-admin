-- 批量建账改为普通模式逐组独立事务后，批次可能部分成功，批次头的不可变汇总列没有稳定含义。
-- 批次头只保留身份字段；费用数、账单数与本位币合计改为读取时按 finance_bills.batch_id 聚合，
-- 按本位币分组，不跨本位币相加。仅删除列，不改动账单与账单行数据。
-- 约束 finance_bill_batch_counts_check 只引用 fee_count、bill_count，随列删除由 PostgreSQL 一并移除。
ALTER TABLE "finance_bill_batches"
  DROP COLUMN "fee_count",
  DROP COLUMN "bill_count",
  DROP COLUMN "total_base_amount",
  DROP COLUMN "base_currency";
