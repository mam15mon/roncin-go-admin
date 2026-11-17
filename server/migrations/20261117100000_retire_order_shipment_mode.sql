-- 根据退役前后 Ent OrdersColumns 差异生成：仅移除 shipment_mode 列。
-- 保留普通订单及主分单关联。
ALTER TABLE "orders" DROP COLUMN "shipment_mode";
