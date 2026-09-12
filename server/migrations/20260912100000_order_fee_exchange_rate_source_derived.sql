-- 汇率交叉套算：直连行缺失时经总部基准币推导的汇率在费用快照标记 DERIVED，
-- 与账单/流水/发票的来源口径对齐（财务仍只维护 X → 基准币 单边行）。
-- DROP 带 IF EXISTS：开发库存在早期建库漂移（该 CHECK 可能缺失），
-- 完整迁移链冷启动下原约束存在时同样无副作用。
ALTER TABLE "order_fees"
  DROP CONSTRAINT IF EXISTS "order_fees_exchange_rate_source_check",
  ADD CONSTRAINT "order_fees_exchange_rate_source_check"
    CHECK ("exchange_rate_source" IN ('SYSTEM', 'BASE_CURRENCY', 'MANUAL', 'DERIVED'));
