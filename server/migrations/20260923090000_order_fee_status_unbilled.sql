-- 订单费用状态简化：取消费用确认环节（09-23-order-fee-bulk-actions 阶段一）。
-- 历史值 DRAFT（草稿）与 CONFIRMED（已确认）按用户决策合并为 UNBILLED（未建账）：
-- 费用保存后即可维护并进入建账候选，不再区分确认前后的中间状态；该区分不保留
-- 回退分支。处理顺序为先放宽约束、再转历史数据、最后收紧约束，行、金额与版本
-- 全部保持不变。

-- 1) 放宽：移除旧状态 CHECK，允许写入目标值 UNBILLED。
ALTER TABLE "order_fees" DROP CONSTRAINT "order_fees_status_check";

-- 2) 转数据：存量草稿与已确认费用统一归并为未建账。
UPDATE "order_fees" SET "status" = 'UNBILLED' WHERE "status" IN ('DRAFT', 'CONFIRMED');

-- 3) 收紧：默认状态改为 UNBILLED，状态域收紧为未建账/已建账/已作废，
--    与 Ent Schema 的 order_fees_status_check 注解同名同表达式。
ALTER TABLE "order_fees" ALTER COLUMN "status" SET DEFAULT 'UNBILLED';
ALTER TABLE "order_fees"
  ADD CONSTRAINT "order_fees_status_check"
  CHECK ("status" IN ('UNBILLED', 'BILLED', 'CANCELLED'));
