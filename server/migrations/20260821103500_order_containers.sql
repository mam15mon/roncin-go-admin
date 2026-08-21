-- 订单集装箱表；结构与 ent/schema/order_container.go 初始定义保持一致。
CREATE TABLE "order_containers" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "container_no" character varying(64) NOT NULL,
  "container_spec_id" uuid NOT NULL,
  "seal_no" character varying(64) NULL,
  "gross_weight_kg" double precision NOT NULL,
  "volume_cbm" double precision NOT NULL,
  "note" character varying(500) NULL,
  "order_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_containers_orders_containers"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);

CREATE INDEX "ordercontainer_updated_at"
  ON "order_containers" ("updated_at");

CREATE UNIQUE INDEX "ordercontainer_order_id_container_no"
  ON "order_containers" ("order_id", "container_no");

CREATE INDEX "ordercontainer_order_id"
  ON "order_containers" ("order_id");

CREATE INDEX "ordercontainer_container_spec_id"
  ON "order_containers" ("container_spec_id");