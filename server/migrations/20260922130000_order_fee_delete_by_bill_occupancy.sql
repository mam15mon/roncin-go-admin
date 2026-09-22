-- 订单费用按账单占用关系删除（09-22-order-fee-delete-by-bill-occupancy）：
-- 1. 账单行来源费用可空：费用物理删除后历史（含已取消账单）行由外键置空，
--    快照字段（费用代码/名称/数量/单价/税额/币种等）继续支撑历史展示；
--    活动账单行不受影响——删除前服务端在事务内复核占用并拒绝。
-- 2. 费用标签关联随费用删除级联清理。
-- 不清理既有 CANCELLED 费用行（未获数据操作授权）。

ALTER TABLE "finance_bill_lines" ALTER COLUMN "order_fee_id" DROP NOT NULL;

ALTER TABLE "finance_bill_lines" DROP CONSTRAINT "finance_bill_lines_order_fees_finance_bill_lines";
ALTER TABLE "finance_bill_lines"
  ADD CONSTRAINT "finance_bill_lines_order_fees_finance_bill_lines"
  FOREIGN KEY ("order_fee_id") REFERENCES "order_fees" ("id") ON DELETE SET NULL;

ALTER TABLE "order_fee_enterprise_tags" DROP CONSTRAINT "order_fee_enterprise_tags_order_fees_enterprise_tag_links";
ALTER TABLE "order_fee_enterprise_tags"
  ADD CONSTRAINT "order_fee_enterprise_tags_order_fees_enterprise_tag_links"
  FOREIGN KEY ("order_fee_id") REFERENCES "order_fees" ("id") ON DELETE CASCADE;
