-- 订单新增危险品包装类别（IATA DGR Packing Group）：nil 兼「未填 / 不适用」
-- 双语义（Class 1/2/7 无包装类别），列形与 shipment_type 一致。
ALTER TABLE "orders" ADD COLUMN "dangerous_packing_group" character varying NULL;
