-- 为订单全量筛选补充可持久化的收发货人、锁定、分享和标签属性。
ALTER TABLE "orders"
  ADD COLUMN "shipper_short_name" varchar(200),
  ADD COLUMN "consignee_short_name" varchar(200),
  ADD COLUMN "locked_at" timestamptz,
  ADD COLUMN "is_shared" boolean NOT NULL DEFAULT false,
  ADD COLUMN "tags" jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD CONSTRAINT "orders_tags_array_check" CHECK (jsonb_typeof("tags") = 'array');

CREATE INDEX "order_organization_id_carrier_id" ON "orders" ("organization_id", "carrier_id");
CREATE INDEX "order_organization_id_origin_location_id" ON "orders" ("organization_id", "origin_location_id");
CREATE INDEX "order_organization_id_destination_location_id" ON "orders" ("organization_id", "destination_location_id");
CREATE INDEX "order_organization_id_locked_at" ON "orders" ("organization_id", "locked_at");
CREATE INDEX "order_organization_id_is_shared" ON "orders" ("organization_id", "is_shared");
CREATE INDEX "order_tags_gin" ON "orders" USING gin ("tags");
