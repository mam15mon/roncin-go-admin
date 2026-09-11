-- 移除订单级运输条款：该字段与提单正文 transport_terms 语义重复且无业务消费方，
-- 按决策完整删除订单模型中的 loading_terms，列数据直接丢弃，不迁移到提单正文。
ALTER TABLE "orders" DROP COLUMN "loading_terms";
