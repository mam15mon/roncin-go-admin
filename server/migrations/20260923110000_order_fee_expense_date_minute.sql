-- 订单费用发生日期分钟精度落库（09-23-order-fee-bulk-actions 阶段二）。
-- 发生日期校验与前端提交已按分钟精度（YYYY-MM-DD HH:mm，16 字符）传递，
-- 而费用与补录快照表的 expense_date 列仍是 varchar(10)，正式迁移链冷启动的
-- 环境无法存储分钟精度（单条保存即失败）。本迁移同步放宽带宽至 varchar(16)；
-- 存量 10 字符纯日期值不受影响，exchange_rate_date 恒存日期部分（10 字符）
-- 保持不变。

ALTER TABLE "order_fees" ALTER COLUMN "expense_date" TYPE character varying(16);
ALTER TABLE "order_fee_supplement_requests" ALTER COLUMN "expense_date" TYPE character varying(16);
