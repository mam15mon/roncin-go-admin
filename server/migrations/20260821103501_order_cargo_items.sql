-- 订单货物明细表；结构与 ent/schema/order_cargo_item.go 保持一致。
CREATE TABLE "order_cargo_items" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "cargo_name" character varying(200) NOT NULL,
  "package_count" bigint NOT NULL,
  "gross_weight_kg" double precision NOT NULL,
  "volume_cbm" double precision NOT NULL,
  "net_weight_kg" double precision NULL,
  "note" character varying(500) NULL,
  "order_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_cargo_items_orders_cargo_items"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);

CREATE INDEX "ordercargoitem_updated_at"
  ON "order_cargo_items" ("updated_at");

CREATE INDEX "ordercargoitem_order_id"
  ON "order_cargo_items" ("order_id");
